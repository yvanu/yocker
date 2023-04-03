package network

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"
)

const ipamDefaultAllocatorPath = "/var/run/yocker/network/ipam/subnet.json"

type IPAM struct {
	// 分配文件存放的位置
	SubnetAllocatorPath string
	// key是网段 value是分为的位图数组
	Subnets map[string]string
}

var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
	Subnets:             make(map[string]string),
}

func (i *IPAM) load() error {
	if _, err := os.Stat(i.SubnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	subnetFile, err := os.Open(i.SubnetAllocatorPath)
	defer subnetFile.Close()
	if err != nil {
		return err
	}
	subnetJson, err := ioutil.ReadAll(subnetFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(subnetJson, &i.Subnets)
	if err != nil {
		logrus.Errorf("序列化子网失败 %v", err)
		return err
	}
	return nil
}

func (i *IPAM) dump() error {
	ipamFileDir, _ := path.Split(i.SubnetAllocatorPath)
	if _, err := os.Stat(ipamFileDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(ipamFileDir, 0644)
		} else {
			return err
		}
	}
	subnetFile, err := os.OpenFile(i.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	defer subnetFile.Close()
	if err != nil {
		return err
	}
	ipamJson, err := json.Marshal(i.Subnets)
	if err != nil {
		return err
	}
	_, err = subnetFile.Write(ipamJson)
	if err != nil {
		return err
	}
	return nil
}

// 获取一个可用的ip
func (i *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	// 加载网段信息
	err = i.load()
	if err != nil {
		logrus.Errorf("加载网段信息失败 %v", err)
	}
	// 返回的网络位长度和总长度 如127.0.0.0/8 伐木机哦8和32
	one, size := subnet.Mask.Size()
	// 如果之前没分配过这个网段 则初始化该网段
	if _, exist := i.Subnets[subnet.String()]; !exist {
		// 用0填满网段 1<<uint8(size-one)标识该网络有多少可用的地址 等价2^(size-one)
		i.Subnets[subnet.String()] = strings.Repeat("0", 1<<uint8(size-one))
	}
	//fmt.Println(i.Subnets[subnet.String()])

	for c, _ := range i.Subnets[subnet.String()] {
		// 找到数组中为0的位置
		if i.Subnets[subnet.String()][c] == '0' {
			// 设置该位置为1 标识该位置的ip被分配了
			// 字符串不能直接修改 需要先转化成bytes数组 之后再转回来
			ipalloc := []byte(i.Subnets[subnet.String()])
			ipalloc[c] = '1'
			i.Subnets[subnet.String()] = string(ipalloc)
			// 这里的ip是网段的初始化ip 192.168.0.0/16 则是192.168.0.0
			ip = subnet.IP.To4()
			// 通过偏移得到分配的ip
			// >> 相当于不要后面几位 1>>2 --> 001>>2 --> 0(01) -->0
			// IP是一个uint的组数 通过数组中每一项加所需的值得到最终ip
			// 比如网段是172.16.0.0/12 要分配的值转化成数是65555
			// 那么[172,16,0,0]上依次加[uint8(65555>>24), uint8(65555>>16), uint(65555>>8), uint(65555>>0)]
			// 即[0,1,0,19] --> 172.17.0.19
			for t := uint(4); t > 0; t -= 1 {
				ip[4-t] += uint8(c >> ((t - 1) * 8))
			}
			ip[3] += 1
			break
		}
	}

	i.dump()
	return
}

func (i *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	err := i.load()
	if err != nil {
		return fmt.Errorf("释放ip失败 %v", err)
	}
	// 计算ip地址在网段位图数组中的索引位置
	c := 0
	// 把ip转成4个字节的表示  172.16.10.168
	releaseIP := ipaddr.To4()
	// 由于ip从1开始分配 所以转换成索引应减1
	releaseIP[3] -= 1
	subnet.IP = subnet.IP.To4()
	for t := uint(4); t > 0; t -= 1 {
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}
	ipalloc := []byte(i.Subnets[subnet.String()])
	ipalloc[c] = '0'
	i.Subnets[subnet.String()] = string(ipalloc)
	i.dump()
	return nil
}
