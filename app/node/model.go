package node

import (
	"time"

	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/sirupsen/logrus"

	"github.com/thank243/lightsailMon/common/ddns"
	"github.com/thank243/lightsailMon/common/notify"
)

type Node struct {
	Svc        *lightsail.Lightsail
	Timeout    time.Duration
	DdnsClient ddns.Client
	Notifier   notify.Notify
	Logger     *logrus.Entry

	name    string
	network string
	ip      string
	port    int
	domain  string
}
