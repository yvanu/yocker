简易版docker

# 当前已实现功能

- [x] run 运行一个容器，支持终端运行，挂载文件，后台运行，指定容器名，指定镜像名，指定环境变量，指定要加入的网络，端口映射
- [x] stop 停止容器
- [x] rm 删除容器
- [x] network 创建网络 目前只支持bridge类型
- [x] log 查看容器日志
- [x] ps 列出所有容器
- [x] exec 进入容器
- [x] commit 把容器打包成镜像
# 未修复bug

- [ ] 容器状态流转bug
- [ ] 容器内进程执行完后状态不变
- [ ] 容器被意外终止，状态不变
- [ ] -it打开的容器 shell被退出 状态不变
- [ ] ...
# todo

- [ ] 实现跨节点的容器互联
- [ ] 使用cgroup进行资源限制
- [ ] 实现host和none类型网络
- [ ] 实现cp命令
- [ ] 实现images命令
- [ ] 实现rmi命令
- [ ] 实现restart命令
- [ ] 实现inspect命令
- [ ] 优化ps命令输出
- [ ] 日志打印优化
- [ ] 代码架构优化
- [ ] ...

# let's try
## 编译
```
go build yocker main/main.go
```
## 准备镜像
```
docker pull busybox 
docker run -d busybox top -b 
docker export - o busybox.tar 容器 ID
mkdir -p /opt/yocker
tar -xvf busybox . tar -C /opt/yocker/busybox/
```
## 创建网络
```
./yocker network create -driver bridge -subnet 10.10.0.1/16 demonw
```
## 运行容器
```
./yocker run -ti -name democ -image busybox -p 8000:8000 -net demonw sh
```
