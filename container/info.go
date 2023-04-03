package container

import (
	"encoding/json"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

type ContainerInfo struct {
	Pid         string   `json:"pid"`
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Command     string   `json:"command"`
	CreateTime  string   `json:"create_time"`
	Status      string   `json:"status"`
	Volume      string   `json:"volume"`
	PortMapping []string `json:"port_mapping"` // todo 待使用
}

func RecordContainerInfo(containerPid int, cmdArr []string, containerName, volume string) (*ContainerInfo, error) {
	uid, _ := uuid.NewV4()
	id := uid.String()
	createTime := time.Now().Format("2006-01-02 15:04:05")
	cmd := strings.Join(cmdArr, "")
	if containerName == "" {
		containerName = id
	}
	cInfo := &ContainerInfo{
		Id:         id,
		Pid:        strconv.Itoa(containerPid),
		Name:       containerName,
		Command:    cmd,
		CreateTime: createTime,
		Status:     Running,
		Volume:     volume,
	}

	jsonBytes, err := json.Marshal(cInfo)
	if err != nil {
		logrus.Errorf("生成容器信息失败 %v", err)
		return nil, err
	}
	jsonStr := string(jsonBytes)
	dirUrl := fmt.Sprintf(DefaultInfoLocation, containerName)
	if err := os.MkdirAll(dirUrl, 0777); err != nil {
		logrus.Errorf("创建容器信息文件失败 %v", err)
		return nil, err
	}
	fileName := dirUrl + "/" + ConfigName
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		logrus.Errorf("创建容器信息文件失败 %v", err)
		return nil, err
	}
	if _, err := file.WriteString(jsonStr); err != nil {
		logrus.Errorf("写入容器信息失败 %v", err)
		return nil, err
	}
	return cInfo, nil
}

func DeleteContainerInfo(containerName string, volume string) error {
	dirUrl := fmt.Sprintf(DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirUrl); err != nil {
		logrus.Errorf("删除容器信息文件失败 %v", err)
		return err
	}
	return nil
}

func GetContainerInfoByName(containerName string) (*ContainerInfo, error) {
	dirURL := fmt.Sprintf(DefaultInfoLocation, containerName)
	configFilePath := dirURL + ConfigName
	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		logrus.Errorf("获取容器信息失败 %v", err)
		return nil, err
	}
	var containerInfo ContainerInfo
	err = json.Unmarshal(contentBytes, &containerInfo)
	if err != nil {
		logrus.Errorf("序列化容器信息失败 %v", err)
		return nil, err
	}
	return &containerInfo, nil
}
