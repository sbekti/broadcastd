package main

import (
	"github.com/labstack/gommon/log"
	"github.com/sbekti/broadcastd/broadcast"
	"os"
	"os/signal"
)

func main() {
	c, err := broadcast.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

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
