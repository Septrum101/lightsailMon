package controller

import (
	"time"

	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/robfig/cron/v3"
)

type Server struct {
	running     bool
	internal    int
	timeout     int
	nodes       []*node
	cron        *cron.Cron
	cronRunning bool
}

type node struct {
	address      string
	port         int
	lastChangeIP time.Time
	svc          *lightsail.Lightsail
}
