package command

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"text/tabwriter"
	"yocker/container"
)

var ListCommand = &cli.Command{
	Name:  "ps",
	Usage: "查看所有容器",
	Action: func(context *cli.Context) error {
		listContainers()
		return nil
	},
}

func listContainers() {
	// dirURL /var/run/yocker
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, "")
	dirURL = dirURL[:len(dirURL)-1]

	files, err := ioutil.ReadDir(dirURL)
	if err != nil {
		logrus.Errorf("读取容器文件目录失败 %v", err)
		return
	}
	var containers []*container.ContainerInfo
	for _, file := range files {
		tmpContainer, err := getContainerInfo(file)
		if err != nil {
			logrus.Errorf("读取容器信息失败 %v", err)
			continue
		}
		containers = append(containers, tmpContainer)
	}

	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	for _, item := range containers{
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreateTime)
	}
	if err := w.Flush(); err != nil{
		logrus.Errorf("flush失败 %v", err)
		return
	}
}

func getContainerInfo(file os.FileInfo) (*container.ContainerInfo, error) {
	if !file.IsDir(){
		return nil, fmt.Errorf("非目录文件 %s", file.Name())
	}
	containerName := file.Name()
	configFileDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFileDir = configFileDir + container.ConfigName

	content, err := ioutil.ReadFile(configFileDir)
	if err != nil{
		logrus.Errorf("读取容器文件失败 %v", err)
		return nil, err
	}
	var containerInfo container.ContainerInfo
	err = json.Unmarshal(content, &containerInfo)
	if err != nil{
		logrus.Errorf("序列化容器信息失败")
		return nil, err
	}
	return &containerInfo, nil
}
