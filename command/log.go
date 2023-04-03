package command

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"yocker/container"
)

var LogCommand = &cli.Command{
	Name:  "log",
	Usage: "获取容器日志",
	Action: func(context *cli.Context) error {
		if context.NArg() < 1 {
			logrus.Errorf("缺少容器名")
			return errors.New("缺少容器名")
		}
		containerName := context.Args().Get(0)
		logContainer(containerName)
		return nil
	},
}

func logContainer(containerName string) {
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	logFileLocation := dirURL + container.ContainerLogFile
	file, err := os.Open(logFileLocation)
	defer file.Close()
	if err != nil {
		logrus.Errorf("打开日志文件失败 %s %v", containerName, err)
		return
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		logrus.Errorf("读取日志文件失败 %s %v", containerName, err)
		return
	}
	fmt.Fprintf(os.Stdout, string(content))
}
