package controller

import (
	"sync"

	"github.com/robfig/cron/v3"

	"github.com/thank243/lightsailMon/app/node"
	"github.com/thank243/lightsailMon/config"
)

type Service struct {
	conf  *config.Config
	nodes []*node.Node
	cron  *cron.Cron
	wg    sync.WaitGroup

	running  bool
	internal int
	timeout  int
	worker   chan bool
}
