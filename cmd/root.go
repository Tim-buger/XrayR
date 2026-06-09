package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/XrayR-project/XrayR/panel"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use: "XrayR",
		Run: func(cmd *cobra.Command, args []string) {
			// 默认动作：启动主流程
			if err := run(); err != nil {
				log.Fatal(err)
			}
		},
	}
)

func init() {
	// 允许通过 -c/--config 指定配置文件
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Config file for XrayR.")
}

func getConfig() *viper.Viper {
	config := viper.New()

	// 根据命令行参数确定配置文件的目录、文件名和扩展名。
	if cfgFile != "" {
		// 有指定配置文件时，按指定路径加载
		configName := path.Base(cfgFile)
		configFileExt := path.Ext(cfgFile)
		configNameOnly := strings.TrimSuffix(configName, configFileExt)
		configPath := path.Dir(cfgFile)
		config.SetConfigName(configNameOnly)
		config.SetConfigType(strings.TrimPrefix(configFileExt, "."))
		config.AddConfigPath(configPath)
		// 让 xray-core 在配置文件目录中查找 geoip.dat、geosite.dat 等资源。
		os.Setenv("XRAY_LOCATION_ASSET", configPath)
		os.Setenv("XRAY_LOCATION_CONFIG", configPath)
	} else {
		// 未指定时，默认当前目录下的 config.yml
		config.SetConfigName("config")
		config.SetConfigType("yml")
		config.AddConfigPath(".")

	}

	if err := config.ReadInConfig(); err != nil {
		// 配置读取失败直接 panic
		log.Panicf("Config file error: %s \n", err)
	}

	// 监听配置文件变更（后续热重载）
	config.WatchConfig()

	return config
}

func run() error {
	// 打印版本信息
	showVersion()

	config := getConfig()
	// 配置映射到 panel.Config（面板对接核心配置）
	panelConfig := &panel.Config{}
	if err := config.Unmarshal(panelConfig); err != nil {
		return fmt.Errorf("Parse config file %v failed: %s \n", cfgFile, err)
	}

	if panelConfig.LogConfig.Level == "debug" {
		// debug 模式输出调用栈
		log.SetReportCaller(true)
	}

	// 创建面板实例（负责与面板通信、启动节点）
	p := panel.New(panelConfig)
	lastTime := time.Now()
	config.OnConfigChange(func(e fsnotify.Event) {
		// 防抖：短时间内只处理一次变更
		// 文件写入可能连续触发多个事件，3 秒内只执行一次热重载。
		if time.Now().After(lastTime.Add(3 * time.Second)) {
			fmt.Println("Config file changed:", e.Name)
			// 停止旧实例
			p.Close()
			// 释放旧 core 配置占用的对象。
			runtime.GC()
			if err := config.Unmarshal(panelConfig); err != nil {
				log.Panicf("Parse config file %v failed: %s \n", cfgFile, err)
			}

			if panelConfig.LogConfig.Level == "debug" {
				log.SetReportCaller(true)
			}

			// 启动新实例
			p.Start()
			lastTime = time.Now()
		}
	})

	// 启动服务
	p.Start()
	defer p.Close()

	// 配置加载完成后主动回收临时对象。
	runtime.GC()
	// 阻塞等待退出信号
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-osSignals

	return nil
}

func Execute() error {
	// cobra 入口
	return rootCmd.Execute()
}
