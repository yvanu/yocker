package command

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os"
	"yocker/container"
)

var RemoveCommand = &cli.Command{
	Name:  "rm",
	Usage: "删除容器",
	Action: func(context *cli.Context) error {
		if context.NArg() < 1 {
			logrus.Errorf("缺少容器名")
			return errors.New("缺少容器名")
		}
		containerName := context.Args().Get(0)
		removeContainer(containerName)
		return nil
	},
}

func removeContainer(containerName string) {
	containerInfo, err := container.GetContainerInfoByName(containerName)
	if err != nil {
		logrus.Errorf("获取容器信息失败 %s %v", containerName, err)
		return
	}
	if containerInfo.Status != container.Stop {
		logrus.Errorf("无法删除非停止状态的容器")
		return
	}
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirURL); err != nil{
		logrus.Errorf("删除容器失败 %s %v", dirURL, err)
		return
	}
}
