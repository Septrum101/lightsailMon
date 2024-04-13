package controller

import (
	"sync"

	"github.com/robfig/cron/v3"

	"github.com/Septrum101/lightsailMon/app/node"
	"github.com/Septrum101/lightsailMon/config"
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
	isIpv6   bool
}
