package controller

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lightsail"
	log "github.com/sirupsen/logrus"
)

func (n *node) changeIP() {
	// attach IP
	log.Debugf("[%s:%d] Attach static IP", n.domain, n.port)
	if _, err := n.svc.AttachStaticIp(&lightsail.AttachStaticIpInput{
		InstanceName: aws.String(n.name),
		StaticIpName: aws.String("LightsailMon"),
	}); err != nil {
		log.Error(err)
	}

	// detach IP
	log.Debugf("[%s:%d] Detach static IP", n.domain, n.port)
	if _, err := n.svc.DetachStaticIp(&lightsail.DetachStaticIpInput{
		StaticIpName: aws.String("LightsailMon"),
	}); err != nil {
		log.Error(err)
	}

	// update node IP
	inst, err := n.svc.GetInstance(&lightsail.GetInstanceInput{InstanceName: aws.String(n.name)})
	if err != nil {
		log.Error(err)
		return
	}
	n.ip = aws.StringValue(inst.Instance.PublicIpAddress)

	// Update domain record
	n.ddnsClient.AddUpdateDomainRecords(n.network, n.ip)
}
