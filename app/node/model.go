package node

import (
	"time"

	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/sirupsen/logrus"

	"github.com/Septrum101/lightsailMon/common/ddns"
	"github.com/Septrum101/lightsailMon/common/notify"
)

type Node struct {
	Network    string
	Svc        *lightsail.Lightsail
	Timeout    time.Duration
	DdnsClient ddns.Client
	Notifier   notify.Notify
	Logger     *logrus.Entry

	name   string
	ip     string
	port   int
	domain string
}
