package controller

import (
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/robfig/cron/v3"

	"github.com/thank243/lightsailMon/app/dns"
)

type Server struct {
	running     bool
	internal    int
	timeout     int
	nodes       []*node
	cron        *cron.Cron
	cronRunning atomic.Bool
}

type node struct {
	network      string
	address      string
	port         int
	lastChangeIP time.Time
	svc          *lightsail.Lightsail
	nameserver   *dns.DoHClient
}
