package node

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestCheckConnection(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	n := &Node{
		network: "tcp4",
		ip:      "127.0.0.1",
		port:    8080,
		timeout: time.Second * 5,
	}
	t.Log(n.checkConnection())
}
