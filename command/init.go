package command

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

var InitCommand = &cli.Command{
	Name:  "init",
	Usage: "内部方法，在容器内运行用户进程",
	Action: func(context *cli.Context) error {
		logrus.Infof("开始初始化")
		cmd := context.Args().Get(0)
		logrus.Infof("命令是%s", cmd)
		err := runContainerInitProcess()
		return err
	},
}

func runContainerInitProcess() error {
	cmdArr := ReadUserCommand()
	if cmdArr == nil || len(cmdArr) == 0 {
		return errors.New("获取用户命令失败")
	}

	SetUpMount()

	path, err := exec.LookPath(cmdArr[0])
	if err != nil {
		logrus.Errorf("获取命令的绝对路径失败 %v", err)
		return nil
	}

	if err := syscall.Exec(path, cmdArr[0:], os.Environ()); err != nil {
		logrus.Errorf(err.Error())
	}
	return nil
}

func ReadUserCommand() []string {
	pipe := os.NewFile(uintptr(3), "pipe")
	msg, err := ioutil.ReadAll(pipe)
	if err != nil {
		logrus.Errorf("从pipe中读取失败 %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}

func SetUpMount() {
	pwd, err := os.Getwd()
	if err != nil {
		logrus.Errorf("获取当前工作目录失败 %v", err)
		return
	}
	logrus.Infof("当前工作目录 %s", pwd)
	err = PivotRoot(pwd)
	if err != nil {
		logrus.Errorf("privot root 失败 %v", err)
		return
	}
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	err = syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		logrus.Errorf("挂载proc到容器失败 %v", err)
		return
	}
	err = syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
	if err != nil {
		logrus.Errorf("挂载tmpfs到容器失败 %v", err)
		return
	}
}

func PivotRoot(root string) error {

	// 要求不能是同一文件系统
	//err := exec.Command("mount", "--make-rprivate", "/").Run()
	err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	if err != nil {
		logrus.Errorf("初始化挂载失败")
		return err
	}

	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("挂载rootfs到它本身失败 %v", err)
	}
	pivotDir := filepath.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return err
	}

	// pivot_root 到新的rootfs
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("pivot_root 失败 %v", err)
	}

	// 修改当前的工作目录到根目录
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / 失败 %v", err)
	}
	pivotDir = filepath.Join("/", ".pivot_root")
	// 取消挂载
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("取消挂载pivot dir 失败 %v", err)
	}
	return os.Remove(pivotDir)
	//return nil
}
