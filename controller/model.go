package controller

import (
	"github.com/go-resty/resty/v2"
	"sync"

	"github.com/robfig/cron/v3"

	"github.com/Septrum101/lightsailMon/app/node"
	"github.com/Septrum101/lightsailMon/config"
)

type Service struct {
	conf     *config.Config
	nodes    []*node.Node
	cron     *cron.Cron
	wg       sync.WaitGroup
	cli      *resty.Client
	running  bool
	internal int
	timeout  int
	worker   chan bool
	isIpv6   bool
}
