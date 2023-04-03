package fs

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
)

const (
	RootUrl         = "/opt/yocker/"
	lowerDirFormat  = "/opt/yocker/%s"
	upperDirFormat  = "/opt/yocker/%s/upper/"
	workDirFormat   = "/opt/yocker/%s/work/"
	mergedDirFormat = "/opt/yocker/%s/merged/"
)

func getImage(imageName string) string {
	return RootUrl + imageName + ".tar"
}

func getUnTar(imageName string) string {
	return RootUrl + imageName + "/"
}

func getLower(imageName string) string {
	return fmt.Sprintf(lowerDirFormat, imageName)
}

func getUpper(containerName string) string {
	return fmt.Sprintf(upperDirFormat, containerName)
}

func getWorker(containerName string) string {
	return fmt.Sprintf(workDirFormat, containerName)
}

func getMerged(containerName string) string {
	return fmt.Sprintf(mergedDirFormat, containerName)
}

func GetUnTar(imageName string) string {
	return RootUrl + imageName + "/"
}

func GetImage(imageName string) string {
	return RootUrl + imageName + ".tar"
}

func GetMerged(containerName string) string {
	return fmt.Sprintf(mergedDirFormat, containerName)
}

func NewWorkSpace(imageName, containerName, volume string) {
	CreateReadOnlyLayer(imageName)
	CreateWriteLayer(containerName)
	CreateMountPoint(containerName, imageName)
	// 判断用户是否执行挂载操作
	if volume != "" {
		volumeURLs := strings.Split(volume, ":")
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			MountVolume(containerName, volumeURLs)
		} else {
			logrus.Errorf("挂载格式不正确")
		}
	}
}

func MountVolume(containerName string, volumeURLs []string) {
	// 创建宿主机文件目录
	hostUrl, containerUrl := volumeURLs[0], volumeURLs[1]
	if err := os.MkdirAll(hostUrl, 0777); err != nil {
		logrus.Errorf("创建宿主机文件目录%s失败: %v", hostUrl, err)
		return
	}
	// 在容器中创建挂载点
	mntURL := getMerged(containerName)
	containerVolumeURL := mntURL + containerUrl
	if err := os.MkdirAll(containerVolumeURL, 0777); err != nil {
		logrus.Errorf("创建容器文件目录%s失败: %v", containerVolumeURL, err)
		return
	}

	// 把宿主机文件目录挂载到容器挂载点
	cmd := exec.Command("mount", "-o", "bind", hostUrl, containerVolumeURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("挂载目录失败 %v", err)
	}
}

func CreateMountPoint(containerName, imageName string) {
	mntURL := getMerged(containerName)
	// // mount -t overlay overlay -o lowerdir=lower1:lower2:lower3,upperdir=upper,workdir=work merged
	if err := os.MkdirAll(mntURL, 0777); err != nil {
		logrus.Errorf("创建 %s 失败 %v", mntURL, err)
		return
	}

	dirs := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", getLower(imageName), getUpper(containerName), getWorker(containerName))
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%v", err)
	}
}

// upper层 work层
func CreateWriteLayer(containerName string) {
	// 参数改成containerName
	upperURL := getUpper(containerName)
	if err := os.MkdirAll(upperURL, 0777); err != nil {
		logrus.Errorf("创建 %s 失败 %v", upperURL, err)
	}

	workURL := getWorker(containerName)
	if err := os.MkdirAll(workURL, 0777); err != nil {
		logrus.Errorf("创建 %s 失败 %v", upperURL, err)
	}
}

// 只读层 lower层
func CreateReadOnlyLayer(imageName string) {
	imageURL := getUnTar(imageName)
	imageTarURL := getImage(imageName)
	exist, err := PathExists(imageURL)
	if err != nil {
		logrus.Errorf("无法判断%s是否存在，%v", imageURL, err)
		return
	}
	if !exist {
		if err := os.MkdirAll(imageURL, 0777); err != nil {
			logrus.Errorf("创建%s 失败 %v", imageURL, err)
			return
		}
		if _, err := exec.Command("tar", "-xvf", imageTarURL, "-C", imageURL).CombinedOutput(); err != nil {
			logrus.Errorf("解压 %s 失败 %v", imageTarURL, err)
			return
		}
	}
}

func PathExists(url string) (bool, error) {
	_, err := os.Stat(url)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func DeleteWorkSpace(containerName, volume string) {
	mntURL := getMerged(containerName)
	rootURL := RootUrl
	if volume != "" {
		volumeURLs := strings.Split(volume, ":")
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			umountVolume(mntURL, volumeURLs)
		}
	}
	DeleteMountPoint(rootURL, mntURL)
	DeleteWriteLayer(rootURL)

}

func umountVolume(mntURL string, volumeURLs []string) {
	// 卸载容器里volume挂载点
	containerUrl := mntURL + volumeURLs[1]
	if _, err := exec.Command("umount", containerUrl).CombinedOutput(); err != nil {
		logrus.Errorf("卸载volume挂载失败")
		return
	}
}

func DeleteWriteLayer(rootURL string) {
	writeURL := rootURL + "upper/"
	if err := os.RemoveAll(writeURL); err != nil {
		logrus.Errorf("删除目录失败 %s error %v", writeURL, err)
	}
	workURL := rootURL + "work"
	if err := os.RemoveAll(workURL); err != nil {
		logrus.Errorf("删除目录失败 %s error %v", workURL, err)
	}
}

func DeleteMountPoint(rootURL string, mntURL string) {
	cmd := exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%v", err)
	}
	if err := os.RemoveAll(mntURL); err != nil {
		logrus.Errorf("删除目录失败 %s error %v", mntURL, err)
	}
}
