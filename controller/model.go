package controller

import (
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/robfig/cron/v3"

	"github.com/thank243/lightsailMon/common/ddns"
	"github.com/thank243/lightsailMon/common/notify"
)

type Service struct {
	sync.RWMutex
	running  bool
	internal int
	timeout  int
	nodes    []*node
	cron     *cron.Cron
	wg       sync.WaitGroup
	worker   chan uint8
}

type node struct {
	name       string
	network    string
	domain     string
	ip         string
	port       int
	svc        *lightsail.Lightsail
	ddnsClient ddns.Client
	notifier   notify.Notify
	timeout    time.Duration
}
