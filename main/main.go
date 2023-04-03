package main

import (
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
	"os"
	"yocker/command"
)

func main() {
	app := &cli.App{
		Name:  "yocker",
		Usage: "simple docker",
		Before: func(context *cli.Context) error {
			logrus.SetFormatter(&logrus.JSONFormatter{})
			logrus.SetOutput(os.Stdout)
			return nil
		},
		Commands: []*cli.Command{
			command.InitCommand,
			command.RunCommand,
			command.CommitCommand,
			command.ListCommand,
			command.LogCommand,
			command.StopCommand,
			command.RemoveCommand,
			command.ExecCommand,
			command.NetworkCommand},
	}
	// 接受os.Args启动程序
	app.Run(os.Args)
}
