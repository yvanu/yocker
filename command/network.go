package command

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"yocker/network"
)

var createCommand = &cli.Command{
	Name:  "create",
	Usage: "创建容器网络",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "driver",
			Usage: "指定驱动",
		},
		&cli.StringFlag{
			Name:  "subnet",
			Usage: "指定子网",
		},
	},
	Action: func(context *cli.Context) error {
		if context.NArg() < 1 {
			fmt.Println("没有输入网络名")
			return fmt.Errorf("没有输入网络名")
		}

		name := context.Args().Get(0)
		driver := context.String("driver")
		subnet := context.String("subnet")
		createNetwork(driver, subnet, name)
		return nil
	},
}

var listCommand = &cli.Command{
	Name:  "list",
	Usage: "查看创建的网络列表",
	Action: func(context *cli.Context) error {
		listNetwork()
		return nil
	},
}

var removeCommand = &cli.Command{
	Name:  "rm",
	Usage: "删除容器网络",
	Action: func(context *cli.Context) error {
		if context.NArg() < 1 {
			logrus.Errorf("没有输入网络名")
			return fmt.Errorf("没有输入网络名")
		}
		networkName := context.Args().Get(0)
		removeNetwork(networkName)
		return nil
	},
}

var NetworkCommand = &cli.Command{
	Name:  "network",
	Usage: "容器网络操作",
	Subcommands: []*cli.Command{
		createCommand,
		listCommand,
		removeCommand,
	},
}

func createNetwork(driverName string, subnet string, name string) {
	err := network.CreateNetwork(driverName, subnet, name)
	if err != nil {
		logrus.Errorf("创建网络失败 %v", err)
		return
	}
}

func listNetwork() {
	network.ListNetwork()
}

func removeNetwork(networkName string) {
	err := network.DeleteNetwork(networkName)
	if err != nil {
		logrus.Errorf("删除网络失败 %v", err)
		return
	}
}
