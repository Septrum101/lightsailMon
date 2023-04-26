package controller

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/robfig/cron/v3"

	"github.com/thank243/lightsailMon/app/ddns"
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
	name         string
	network      string
	domain       string
	ip           string
	port         int
	lastChangeIP time.Time
	svc          *lightsail.Lightsail
	ddnsClient   ddns.Client
}
