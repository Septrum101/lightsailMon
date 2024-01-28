package node

import (
	"time"

	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/sirupsen/logrus"

	"github.com/thank243/lightsailMon/common/ddns"
	"github.com/thank243/lightsailMon/common/notify"
)

type Node struct {
	svc        *lightsail.Lightsail
	name       string
	network    string
	ip         string
	port       int
	domain     string
	timeout    time.Duration
	ddnsClient ddns.Client
	notifier   notify.Notify
	logger     *logrus.Entry
}
