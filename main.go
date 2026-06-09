package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/XrayR-project/XrayR/cmd"
)

func main() {
	// 入口：执行命令行根命令（会加载配置并启动面板对接）
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
