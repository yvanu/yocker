package command

import (
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os/exec"
	"yocker/fs"
)

var CommitCommand = &cli.Command{
	Name:  "commit",
	Usage: "把容器运行状态保存成镜像",
	Action: func(context *cli.Context) error {
		if context.NArg() < 2 {
			logrus.Errorf("缺少镜像名或容器名")
			return errors.New("缺少镜像名或容器名")
		}
		logrus.Infof("开始保存成镜像")
		containerName := context.Args().Get(0)
		imageName := context.Args().Get(1)

		commitContainer(containerName, imageName)
		return nil
	},
}

func commitContainer(containerName, imageName string) {
	//mntURL := "/opt/yocker/yocker/merged"
	//imageTar := "/opt/yocker/yocker/" + imageName + ".tar"

	mntURL := fs.GetMerged(containerName)
	imageTar := fs.GetImage(imageName)

	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {
		logrus.Errorf("压缩到tar失败")
	}
}
