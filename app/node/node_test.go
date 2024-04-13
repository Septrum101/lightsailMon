package node

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestCheckConnection(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	n := &Node{
		Network: "tcp4",
		ip:      "127.0.0.1",
		port:    8080,
		Timeout: time.Second * 5,
	}
	t.Log(n.checkConnection())
}
