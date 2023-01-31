package controller

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lightsail"
	log "github.com/sirupsen/logrus"
)

func (n *node) changeIP(nameMap map[string]string) {
	ips := n.nameserver.LookupIP(n.address)
	for i := range ips {
		if v, ok := nameMap[ips[i]]; ok {
			// attach IP
			log.Debugf("[%s:%d] Attach static IP", n.address, n.port)
			if _, err := n.svc.AttachStaticIp(&lightsail.AttachStaticIpInput{
				InstanceName: aws.String(v),
				StaticIpName: aws.String("LightsailMon"),
			}); err != nil {
				log.Error(err)
			}

			// detach IP
			log.Debugf("[%s:%d] Detach static IP", n.address, n.port)
			if _, err := n.svc.DetachStaticIp(&lightsail.DetachStaticIpInput{
				StaticIpName: aws.String("LightsailMon"),
			}); err != nil {
				log.Error(err)
			}

			return
		}
	}
}
