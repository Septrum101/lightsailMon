package controller

import (
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestCheckConnection(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	n := &node{
		network: "tcp4",
		ip:      "127.0.0.1",
		port:    8080,
	}
	t.Log(n.checkConnection(5))
}
