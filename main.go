package main

import (
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/sbekti/broadcastd/broadcast"
	"os"
	"os/signal"
	"runtime"
	"time"
)

func main() {
	// TODO: Make log level configurable.
	log.SetLevel(log.DEBUG)

	go func() {
		for {
			fmt.Printf("GOROUTINES: %d\n", runtime.NumGoroutine())
			time.Sleep(3 * time.Second)
		}
	}()

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
