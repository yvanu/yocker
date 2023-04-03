package command

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"yocker/container"
	"yocker/fs"
	"yocker/network"
)

var RunCommand = &cli.Command{
	Name:  "run",
	Usage: "在限制命名空间和cgroup的情况下创建一个容器，yocker run -ti [command]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "ti",
			Usage: "是否启用终端",
		},
		&cli.StringFlag{
			Name:  "v",
			Usage: "volume挂载",
		},
		&cli.BoolFlag{
			Name:  "d",
			Usage: "后台运行容器",
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: "容器名",
		},
		&cli.StringFlag{
			Name:  "image",
			Usage: "运行的镜像名",
		},
		&cli.StringSliceFlag{
			Name:  "e",
			Usage: "容器运行的环境变量",
		},
		&cli.StringFlag{
			Name:  "net",
			Usage: "容器要加入的网络",
		},
		&cli.StringSliceFlag{
			Name:  "p",
			Usage: "端口映射",
		},
	},
	Action: func(context *cli.Context) error {
		if context.NArg() < 1 {
			logrus.Errorf("缺少启动命令或镜像名")
			return errors.New("缺少启动命令或镜像名")
		}

		imageName := context.String("image")

		tty := context.Bool("ti")
		detach := context.Bool("d")

		if tty && detach {
			logrus.Errorf("终端运行和后台运行不能共存")
			return fmt.Errorf("终端运行和后台运行不能共存")
		}

		if detach {
			tty = false
		} else {
			tty = true
		}

		containerName := context.String("name")
		volume := context.String("v")
		envArr := context.StringSlice("e")

		networkName := context.String("net")
		portMapping := context.StringSlice("p")

		Run(context.Args().Slice(), tty, volume, containerName, imageName, envArr, networkName, portMapping)
		return nil
	},
}

func Run(cmdArr []string, tty bool, volume, containerName, imageName string, envArr []string, networkName string, portMapping []string) {
	// 先启动一个父进程
	parent, writePipe := NewParentProcess(tty, volume, containerName, imageName, envArr)
	if parent == nil {
		logrus.Errorf("创建父进程失败")
		return
	}
	if err := parent.Start(); err != nil {
		logrus.Error(err)
	}

	containerInfo, err := container.RecordContainerInfo(parent.Process.Pid, cmdArr, containerName, volume)
	if err != nil {
		logrus.Errorf("记录容器信息失败 %v", err)
		return
	}

	if networkName != "" {
		network.Init()
		containerInfo.PortMapping = portMapping
		if err := network.Connect(networkName, containerInfo); err != nil {
			logrus.Errorf("加入网络失败")
			return
		}
	}

	// 发送init命令
	sendInitCommand(cmdArr, writePipe)
	if tty {
		parent.Wait()
		//mntURL := "/opt/yocker/yocker/merged/"
		//rootURL := "/opt/yocker/yocker/"
		fs.DeleteWorkSpace(containerName, volume)
		err := container.DeleteContainerInfo(containerName, volume)
		if err != nil {
			logrus.Errorf("删除容器信息失败 %v", err)
			return
		}
	}
	os.Exit(0)
}

func NewParentProcess(tty bool, volume, containerName, imageName string, envArr []string) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		logrus.Errorf("创建管道失败 %v", err)
		return nil, nil
	}
	args := []string{"init"}
	command := exec.Command("/proc/self/exe", args...)
	//command.Dir = "/opt/yocker/yocker/busybox"
	command.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if tty {
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
	} else {
		// 后台运行 把输出重定向到日志
		dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
		if err := os.MkdirAll(dirURL, 0622); err != nil {
			logrus.Errorf("创建父进程中创建容器目录失败 %s %v", dirURL, err)
			return nil, nil
		}
		stdLogFilePath := dirURL + container.ContainerLogFile
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {
			logrus.Errorf("创建父进程中创建日志文件失败 %v", err)
			return nil, nil
		}
		command.Stdout = stdLogFile
	}
	command.Env = append(os.Environ(), envArr...)
	command.ExtraFiles = []*os.File{readPipe}
	//mntURL := "/opt/yocker/yocker/merged/"
	//rootURL := "/opt/yocker/yocker/"
	fs.NewWorkSpace(imageName, containerName, volume)
	command.Dir = fs.GetUnTar(imageName)
	return command, writePipe
}

func sendInitCommand(cmdArr []string, pipe *os.File) {
	defer pipe.Close()

	cmd := strings.Join(cmdArr, " ")
	logrus.Infof("用户命令是 %s", cmd)
	pipe.WriteString(cmd)
}

func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, err
}
