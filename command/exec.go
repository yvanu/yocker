package command

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"yocker/container"
	_ "yocker/nsenter"
)

const (
	ENV_EXEC_PID = "yocker_pid"
	ENV_EXEC_CMD = "yocker_cmd"
)

var ExecCommand = &cli.Command{
	Name:  "exec",
	Usage: "进入容器",
	Action: func(context *cli.Context) error {
		if os.Getenv(ENV_EXEC_PID) != "" {
			logrus.Infof("已经被设置了 %d %s", os.Getpid(), os.Getenv(ENV_EXEC_CMD))
			return nil
		}
		if context.NArg() < 2 {
			logrus.Errorf("缺少容器名或者命令")
			return errors.New("缺少容器名或者命令")
		}
		containerName := context.Args().Get(0)
		execContainer(context.Args().Slice()[1:], containerName)
		return nil
	},
}

// exec时可以获取到容器的自定义的环境变量
func getEnvByPid(pid string) []string {
	path := fmt.Sprintf("/proc/%s/environ", pid)
	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		logrus.Errorf("读取环境变量失败 %v", err)
		return nil
	}
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs
}

func execContainer(cmdArr []string, containerName string) {
	containerInfo, err := container.GetContainerInfoByName(containerName)
	if err != nil {
		logrus.Errorf("获取容器信息失败 %s %v", containerName, err)
		return
	}
	cmdStr := strings.Join(cmdArr, " ")
	logrus.Infof("进入的容器是 %s 命令是 %s", containerInfo.Name, cmdStr)
	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	os.Setenv(ENV_EXEC_PID, containerInfo.Pid)
	os.Setenv(ENV_EXEC_CMD, cmdStr)

	cmd.Env = append(os.Environ(), getEnvByPid(containerInfo.Pid)...)

	if err := cmd.Run(); err != nil {
		logrus.Errorf("进入容器失败 %s %v", containerInfo.Name, err)
	}
}
