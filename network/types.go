package network

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"os"
	p "path"
)
import "github.com/vishvananda/netlink"

type Network struct {
	Name    string     `json:"name"`
	IpRange *net.IPNet `json:"ip_range"`
	Driver  string     `json:"driver"`
}

func (n *Network) dump(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(path, 0644)
		} else {
			return err
		}
	}
	// 保存的文件名是网络的名字
	nwPath := p.Join(path, n.Name)
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	defer nwFile.Close()
	if err != nil {
		logrus.Errorf("打开文件失败 %v", err)
		return err
	}
	nwJson, err := json.Marshal(n)
	if err != nil {
		logrus.Errorf("序列化网络失败 %v", err)
		return err
	}
	_, err = nwFile.Write(nwJson)
	if err != nil {
		logrus.Errorf("把网络写入文件失败 %v", err)
		return err
	}
	return nil
}

func (n *Network) load(path string) error {
	nwFile, err := os.Open(path)
	defer nwFile.Close()
	if err != nil {
		fmt.Errorf("打开网络文件失败 %v", err)
		return err
	}
	nwJson, err := ioutil.ReadAll(nwFile)
	if err != nil {
		fmt.Errorf("读取网络文件失败 %v", err)
		return err
	}
	err = json.Unmarshal(nwJson, n)
	if err != nil {
		logrus.Errorf("序列化网络失败 %v", err)
		return err
	}
	return nil
}

func (n *Network) remove(path string) error {
	if _, err := os.Stat(p.Join(path, n.Name)); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	} else {
		return os.Remove(p.Join(path, n.Name))
	}
}

type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"device"`
	IPAddress   net.IP           `json:"ip_address"`
	MacAddress  net.HardwareAddr `json:"mac_address"`
	PortMapping []string         `json:"port_mapping"`
	Network     *Network
}

type NetworkDriver interface {
	Name() string
	// Create 创建网络
	Create(subnet string, name string) (*Network, error)
	Delete(network *Network) error
	// Connect 连接网络端点到网络
	Connect(network *Network, endpoint *Endpoint) error
	// DisConnect 从网络上移除网络端点
	DisConnect(network *Network, endpoint *Endpoint) error
}
