package controller

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/robfig/cron/v3"

	"github.com/thank243/lightsailMon/app/dns"
)

type Server struct {
	sync.RWMutex
	running     bool
	internal    int
	timeout     int
	nodes       []*node
	cron        *cron.Cron
	cronRunning atomic.Bool
	wg          sync.WaitGroup
	worker      chan uint8
}

type node struct {
	network      string
	address      string
	port         int
	lastChangeIP time.Time
	svc          svc
	nameserver   *dns.DoHClient
}

type svc struct {
	*sync.RWMutex
	*lightsail.Lightsail
}
