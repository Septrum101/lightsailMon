package controller

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lightsail"
	log "github.com/sirupsen/logrus"

	"github.com/thank243/lightsailMon/app"
)

func (n *node) changeIP(nameMap map[string]string) {

	ips := app.LookupIP(n.address)
	for i := range ips {
		if v, ok := nameMap[ips[i]]; ok {
			// attach IP
			if _, err := n.svc.AttachStaticIp(&lightsail.AttachStaticIpInput{
				InstanceName: aws.String(v),
				StaticIpName: aws.String("LightsailMon"),
			}); err != nil {
				log.Error(err)
				continue
			}

			// detach IP
			if _, err := n.svc.DetachStaticIp(&lightsail.DetachStaticIpInput{
				StaticIpName: aws.String("LightsailMon"),
			}); err != nil {
				log.Error(err)
			}
			n.lastChangeIP = time.Now()
			return
		}
	}
}
