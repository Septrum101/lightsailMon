package controller

import (
	"sync"

	"github.com/robfig/cron/v3"

	"github.com/thank243/lightsailMon/app/node"
	"github.com/thank243/lightsailMon/config"
)

type Service struct {
	sync.RWMutex

	conf     *config.Config
	running  bool
	internal int
	timeout  int
	nodes    []*node.Node
	cron     *cron.Cron
	wg       sync.WaitGroup
	worker   chan bool
}
