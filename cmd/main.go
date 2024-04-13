package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"

	"github.com/Septrum101/lightsailMon/config"
	"github.com/Septrum101/lightsailMon/controller"
)

func main() {
	config.ShowVersion()

	printVersion := flag.Bool("version", false, "show version")
	flag.Parse()
	if *printVersion {
		return
	}

	// init config
	getConfig := config.GetConfig()
	c := new(config.Config)
	if err := getConfig.Unmarshal(c); err != nil {
		log.Panic(err)
	}

	// start service
	s := controller.New(c)
	s.Start()

	// hot reload configure
	lastTime := time.Now()
	getConfig.OnConfigChange(func(e fsnotify.Event) {
		if time.Now().After(lastTime.Add(time.Second * 3)) {
			log.Println("Config file changed:", e.Name)
			if err := getConfig.Unmarshal(c); err != nil {
				log.Panic(err)
			}
			// release server resource
			s.Close()
			s = nil

			// create server
			s = controller.New(c)
			s.Start()
		}
		lastTime = time.Now()
	})
	getConfig.WatchConfig()

	// Running backend
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-osSignals
}
