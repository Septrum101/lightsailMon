package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"

	"github.com/thank243/lightsailMon/config"
	"github.com/thank243/lightsailMon/controller"
)

func main() {
	config.ShowVersion()

	printVersion := flag.Bool("version", false, "show version")
	flag.Parse()
	if *printVersion {
		return
	}

	// init config
	conf := config.GetConfig()
	c := new(config.Config)
	if err := conf.Unmarshal(c); err != nil {
		log.Panic(err)
	}

	// init log level
	if l, err := log.ParseLevel(c.LogLevel); err != nil {
		log.Panic(err)
	} else {
		log.SetLevel(l)
	}

	// start server
	s := controller.New(c)
	s.Start()

	// Hot reload configure
	lastTime := time.Now()
	conf.OnConfigChange(func(e fsnotify.Event) {
		if time.Now().After(lastTime.Add(time.Second * 3)) {
			log.Println("Config file changed:", e.Name)
			if err := conf.Unmarshal(c); err != nil {
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
	conf.WatchConfig()

	// Running backend
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-osSignals
}
