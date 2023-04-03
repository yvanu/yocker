package network

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"io/fs"
	"net"
	"os"
	"os/exec"
	p "path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"yocker/container"
)

var drivers = map[string]NetworkDriver{"bridge": &defaultBridgeDriver}
var networks map[string]*Network = make(map[string]*Network)

const defaultNetworkPath = "/var/run/yocker/network/network"

func CreateNetwork(driver, subnet, name string) error {
	_, cidr, _ := net.ParseCIDR(subnet)
	gatewayIp, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = gatewayIp
	nw, err := drivers[driver].Create(cidr.String(), name)
	fmt.Println(nw)
	if err != nil {
		return err
	}
	// todo 不需要添加到内存吗？？
	//networks[name] = nw
	return nw.dump(defaultNetworkPath)
}

func Connect(networkName string, cinfo *container.ContainerInfo) error {
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("没有该网络 %s", networkName)
	}

	ip, err := ipAllocator.Allocate(network.IpRange)

	if err != nil {
		logrus.Errorf("获取可用ip失败 %v", err)
		return err
	}
	// 创建网络端点
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.Id, networkName),
		IPAddress:   ip,
		Network:     network,
		PortMapping: cinfo.PortMapping,
	}

	err = drivers[network.Driver].Connect(network, ep)
	if err != nil {
		logrus.Errorf("调用驱动设置网络失败 %v", err)
		return err
	}
	// 进入到容器的网络命名空间配置容器网络设备的ip和路由
	// 应该获取网桥的地址然后设置到容器的网关
	gw := getGw(networkName)
	err = configEndpointIpAddressAndRoute(ep, cinfo, gw)
	if err != nil {
		logrus.Errorf("设置容器的ip和路由失败 %v", err)
		return err
	}
	return configPortMapping(ep, cinfo)
}

func getGw(networkName string) *net.IP {
	br, err := net.InterfaceByName(networkName)
	if err != nil {
		logrus.Errorf("获取网桥失败 %v", err)
		return nil
	}
	addrs, err := br.Addrs()
	if err != nil {
		logrus.Errorf("获取网桥ip失败 %v", err)
		return nil
	}
	ipv4, _, _ := net.ParseCIDR(addrs[0].String())
	return &ipv4
}

func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {
	for _, pm := range ep.PortMapping {
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			logrus.Errorf("端口映射格式错误 %s", pm)
			continue
		}

		// 在iptables的PREROUTING中添加DNAT规则
		iptablesCmd := fmt.Sprintf(
			"-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		output, err := cmd.Output()
		if err != nil {
			logrus.Errorf("iptables失败 %v %v", output, err)
			continue
		}
	}
	return nil
}

func configEndpointIpAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo, gw *net.IP) error {
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("获取网络端点失败 %v", err)
	}
	// 将容器的网络端点加入到容器的网络空间中
	// 并且使这个函数下面的操作都在这个网络空间中进行
	// 执行完函数后，恢复到默认的网络康健
	defer enterContainerNetns(&peerLink, cinfo)()

	// 获取到容器的ip地址和网段 用于配置容器内部接口地址
	// 如ip是192.168.1.2 网段是192.168.1.0/24
	// 那么得到的ip字符串是192.168.1.2/24

	interfaceIP := ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress

	// 调用setInterfaceIP函数设置容器内Veth端点的ip
	if err = setInterfaceIp(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("设置网口ip失败 %s %v", ep.Device.PeerName, err)
	}
	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return fmt.Errorf("启动网口失败 %s %v", ep.Device.PeerName, err)
	}
	if err = setInterfaceUP("lo"); err != nil {
		return fmt.Errorf("启动lo网口失败")
	}

	// 设置容器内的外部请求都通过容器内的Veth端点访问
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	// route add -net 0.0.0.0/0 gw(网桥地址) dev(容器内Veth端点设备)
	// gw有问题 应该添加网桥的ip
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		//Gw:        ep.Network.IpRange.IP,
		Gw:        *gw,
		Dst:       cidr,
	}
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return fmt.Errorf("添加路由失败 %v", err)
	}
	return nil
}

func enterContainerNetns(epLink *netlink.Link, cinfo *container.ContainerInfo) func() {
	// 找到容器的Net Namespace
	// /proc/[pid]/ns/net 打开这个文件的文件描述符就可以操作net namespace
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("获取容器的网络命名空间失败 %v", err)
		return nil
	}
	nsFD := f.Fd()

	// 锁定当前程序执行的线程，不锁定操作系统的线程的话goroutine可能会被调度到别的线程上 无法保证一直在所需要的命名空间
	runtime.LockOSThread()
	if err = netlink.LinkSetNsFd(*epLink, int(nsFD)); err != nil {
		logrus.Errorf("把veth放到容器命名空间失败 %v", err)
		return nil
	}

	// 获取当前网络的命名空间 用于后面从容器的明明康健退出
	origns, err := netns.Get()
	if err != nil {
		logrus.Errorf("获取当前命名空间失败 %v", err)
		return nil
	}
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		logrus.Errorf("当前进程加入容器命名空间失败 %v", err)
	}
	return func() {
		netns.Set(origns)
		origns.Close()
		runtime.UnlockOSThread()
		f.Close()
	}

}

// 加载所有网络配置到networks字典中
func Init() error {
	//bridgeDriver := BridgeNetworkDriver{}
	//drivers[bridgeDriver.Name()] = &bridgeDriver
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(defaultNetworkPath, 0644)
		} else {
			return err
		}
	}
	filepath.Walk(defaultNetworkPath, func(nwpath string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		_, nwName := p.Split(nwpath)
		nw := &Network{Name: nwName}
		if err := nw.load(nwpath); err != nil {
			logrus.Errorf("加载网络失败 %v", err)
			return err
		}
		networks[nwName] = nw
		return nil
	})
	return nil
}

func ListNetwork() {
	Init()
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprintf(w, "NAME\tIpRange\tDriver\n")
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			nw.Name,
			nw.IpRange.String(),
			nw.Driver)
	}
	err := w.Flush()
	if err != nil {
		logrus.Errorf("刷新到输出失败 %v", err)
		return
	}
}

func DeleteNetwork(networkName string) error {
	Init()
	nw, ok := networks[networkName]
	fmt.Println(nw)
	if !ok {
		return fmt.Errorf("网络不存在 %s", networkName)
	}
	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return fmt.Errorf("释放ip失败 %v", err)
	}

	// 调用网络驱动删除创建的设备与配置
	if err := drivers[nw.Driver].Delete(nw); err != nil {
		return fmt.Errorf("驱动删除网络失败 %v", err)
	}
	return nw.remove(defaultNetworkPath)
}
