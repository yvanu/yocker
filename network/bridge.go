package network

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os/exec"
	"strings"
)

type BridgeNetworkDriver struct {
	name string
}

var defaultBridgeDriver = BridgeNetworkDriver{name: "bridge"}

func (b *BridgeNetworkDriver) Name() string {
	return b.name
}


func (b *BridgeNetworkDriver) initBridge(n *Network) error {
	bridgeName := n.Name
	// 创建网桥虚拟设备
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("添加网桥失败 %v", err)
	}
	// 设置网桥的地址和路由
	gatewayIP := n.IpRange
	gatewayIP.IP = n.IpRange.IP

	if err := setInterfaceIp(bridgeName, gatewayIP.String()); err != nil {
		return fmt.Errorf("给网桥添加ip失败 %v", err)
	}
	// 启动网桥
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("启动网桥失败 %v", err)
	}
	// 设置iptables的net规则
	if err := setupIPTables(bridgeName, n.IpRange); err != nil {
		return fmt.Errorf("创建iptables的转发规则失败 %v", err)
	}
	return nil
}

func createBridgeInterface(name string) error {
	// 检查是否存在同名bridge
	_, err := net.InterfaceByName(name)
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	// 初始化一个link对象
	la := netlink.NewLinkAttrs()
	la.Name = name

	br := &netlink.Bridge{LinkAttrs: la}
	// ip link add xxx
	err = netlink.LinkAdd(br)
	if err != nil {
		return fmt.Errorf("添加网桥失败 %v", err)
	}
	return nil
}

// setInterfaceIp("testbridge", "192.168.0.1/24")
func setInterfaceIp(name string, ip string) error {
	// 找到网口
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("获取网口失败 %v", err)
	}
	// netlink.ParseIPNet 是对 net.ParseCIDR 的封装
	// ipNet中包含了网段的信息 192.168.0.0/24 和IP 192.168.0.1
	ipNet, err := netlink.ParseIPNet(ip)
	if err != nil {
		return fmt.Errorf("解析ip失败 %v", err)
	}
	//给网口配置地址 ip addr add xxx
	// 如果同时配置了地址所在网段的信息如192.168.0.0/24 还会配置路由表192.168.0.0/24转发到网口上
	addr := &netlink.Addr{IPNet: ipNet, Label: "", Flags: 0, Scope: 0}
	return netlink.AddrAdd(iface, addr)
}

func setInterfaceUP(name string) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("获取网口失败 %s %v", name, err)
	}
	// ip link set xxx up
	err = netlink.LinkSetUp(iface)
	if err != nil {
		return fmt.Errorf("启动网口失败 %s %v", name, err)
	}
	return nil
}

// todo 可能有问题
func setupIPTables(name string, subnet *net.IPNet) error {
	// iptables -t nat -A POSTROUTING -s {subnet} ! -o {deviceName} -j MASQUERADE
	//iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), name)
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s -j MASQUERADE", subnet.String())
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	output, err := cmd.Output()
	if err != nil {
		logrus.Errorf("iptables执行失败 %v %v", output, err)
	}
	return nil
}

func (b *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip
	n := &Network{
		Name:    name,
		IpRange: ipRange,
		Driver:  "bridge",
	}
	err := b.initBridge(n)
	if err != nil {
		logrus.Errorf("初始化网桥失败 %v", err)
		return nil, err
	}
	return n, nil
}

func (b *BridgeNetworkDriver) Delete(network *Network) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	return netlink.LinkDel(br)
}

func (b *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	// 创建Veth接口的配置
	la := netlink.NewLinkAttrs()
	la.Name = endpoint.ID[:5]
	// 把一端挂载到网桥上
	la.MasterIndex = br.Attrs().Index

	// 创建veth对象，通过peername配置veth另外一端的接口名
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}
	// 创建veth接口
	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("创建veth接口失败 %v", err)
	}

	// ping不通是因为网口没UP
	err = setInterfaceUP(endpoint.Device.Name)
	if err != nil {
		logrus.Errorf("启动veth网口失败 %v", err)
		return err
	}
	return nil
}

func (b *BridgeNetworkDriver) DisConnect(network *Network, endpoint *Endpoint) error {
	//bridgeName := network.Name
	//br, err := netlink.LinkByName(bridgeName)
	//if err != nil {
	//	return err
	//}
	//err = netlink.LinkDel(&endpoint.Device)
	//if err != nil {
	//	return err
	//}
	return nil
}

var _ NetworkDriver = new(BridgeNetworkDriver)
