package main

import (
	"time"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/sirupsen/logrus"
	"github.com/soulteary/stargate/src/internal/config"
)

func main() {
	// Display startup banner
	showBanner()

	// Initialize logger
	initLogger()

	// Initialize configuration
	if err := initConfig(); err != nil {
		logrus.Fatal("Failed to initialize config: ", err)
	}

	// Create and start server
	app := createApp()
	if err := startServer(app); err != nil {
		logrus.Fatal("Failed to start web server: ", err)
	}
}

// showBanner displays the startup banner
func showBanner() {
	pterm.DefaultBox.Println(
		putils.CenterText(
			"Stargate\n" +
				"Your Gateway to Secure Microservices\n" +
				"Version: " + Version,
		),
	)
	time.Sleep(time.Millisecond) // Don't ask why, but this fixes the docker-compose log
}

// initLogger initializes the logging system
func initLogger() {
	logrus.SetFormatter(&logrus.TextFormatter{})
}

// initConfig initializes the configuration
func initConfig() error {
	if err := config.Initialize(); err != nil {
		return err
	}

	if config.Debug.ToBool() {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	return nil
}
