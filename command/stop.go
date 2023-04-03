package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"strconv"
	"syscall"
	"yocker/container"
)

var StopCommand = &cli.Command{
	Name:  "stop",
	Usage: "停止容器",
	Action: func(context *cli.Context) error {
		if context.NArg() < 1 {
			logrus.Errorf("缺少容器名")
			return errors.New("缺少容器名")
		}
		containerName := context.Args().Get(0)
		stopContainer(containerName)
		return nil
	},
}

func stopContainer(containerName string) {
	containerInfo, err := container.GetContainerInfoByName(containerName)
	if err != nil {
		logrus.Errorf("获取容器信息失败 %s %v", containerName, err)
		return
	}
	pidInt, err := strconv.Atoi(containerInfo.Pid)
	if err != nil {
		logrus.Errorf("转化容器的pid失败 %v", err)
		return
	}
	if err := syscall.Kill(pidInt, syscall.SIGTERM); err != nil {
		logrus.Errorf("停止容器失败 %s %v", containerName, err)
		return
	}
	containerInfo.Status = container.Stop
	containerInfo.Pid = " "
	newContentBytes, err := json.Marshal(containerInfo)
	if err != nil {
		logrus.Errorf("把停止后的容器信息序列化失败 %s %v", containerName, err)
		return
	}
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName
	if err := ioutil.WriteFile(configFilePath, newContentBytes, 0622); err != nil {
		logrus.Errorf("写入文件失败 %s %v", configFilePath, err)
	}
}
