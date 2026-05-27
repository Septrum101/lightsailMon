package node

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lightsail"
	log "github.com/sirupsen/logrus"

	"github.com/Septrum101/lightsailMon/config"
)

func TestCheckConnection(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	n := &Node{
		Network: "tcp4",
		ip:      "127.0.0.1",
		port:    8080,
		Timeout: time.Second * 5,
		Logger:  log.WithFields(log.Fields{}),
	}
	t.Log(n.checkConnection())
}

func TestDualStack(t *testing.T) {
	c := new(config.Config)
	config.GetConfig().Unmarshal(c)
	nodes := New(c.Nodes[0])
	n := nodes[0]

	n.Svc.GetInstance(context.Background(), &lightsail.GetInstanceInput{InstanceName: aws.String(n.name)})
	n.disableDualStack()
	n.enableDualStack()
}
