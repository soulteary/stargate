package main

import (
	"time"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/sirupsen/logrus"
	"github.com/soulteary/stargate/src/internal/config"
)

func main() {
	// 显示启动横幅
	showBanner()

	// 初始化日志
	initLogger()

	// 初始化配置
	if err := initConfig(); err != nil {
		logrus.Fatal("Failed to initialize config: ", err)
	}

	// 创建并启动服务器
	app := createApp()
	if err := startServer(app); err != nil {
		logrus.Fatal("Failed to start web server: ", err)
	}
}

// showBanner 显示启动横幅
func showBanner() {
	pterm.DefaultBox.Println(
		putils.CenterText(
			"Stargate\n" +
				"limited access",
		),
	)
	time.Sleep(time.Millisecond) // Don't ask why, but this fixes the docker-compose log
}

// initLogger 初始化日志系统
func initLogger() {
	logrus.SetFormatter(&logrus.TextFormatter{})
}

// initConfig 初始化配置
func initConfig() error {
	if err := config.Initialize(); err != nil {
		return err
	}

	if config.Debug.ToBool() {
		logrus.SetLevel(logrus.DebugLevel)
	}

	return nil
}
