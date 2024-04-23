package node

import (
	"github.com/Septrum101/lightsailMon/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lightsail"
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

func TestDualStack(t *testing.T) {
	c := new(config.Config)
	config.GetConfig().Unmarshal(c)
	node := New(c.Nodes[0])
	n := node[0]

	n.Svc.GetInstance(&lightsail.GetInstanceInput{InstanceName: aws.String(n.name)})
	n.disableDualStack()
	n.enableDualStack()
}
