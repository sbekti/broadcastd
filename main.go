package main

import (
	"flag"
	"github.com/sbekti/broadcastd/broadcast"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
)

const (
	defaultConfig = "/etc/broadcastd/config.yaml"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "c", defaultConfig, "path to config file")
	flag.Parse()

	c, err := broadcast.LoadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	level, err := log.ParseLevel(c.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)

	b := broadcast.NewBroadcast(c)

	go func() {
		if err := b.Start(); err != nil {
			log.Info("stopping broadcast")
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit

	if err := b.Stop(); err != nil {
		log.Fatal(err)
	}
}
