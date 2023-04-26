package controller

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lightsail"
	log "github.com/sirupsen/logrus"
)

func (n *node) changeIP() {
	switch n.network {
	case "tcp4":
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
	case "tcp6":
		// disable dualstack network
		log.Debugf("[%s:%d] Disable dualstack network", n.domain, n.port)
		if _, err := n.svc.SetIpAddressTypeRequest(&lightsail.SetIpAddressTypeInput{
			IpAddressType: aws.String("ipv4"),
			ResourceName:  aws.String(n.name),
		}); err != nil {
			log.Error(err)
		}

		// enable dualstack network
		log.Debugf("[%s:%d] Enable dualstack network", n.domain, n.port)
		if _, err := n.svc.SetIpAddressTypeRequest(&lightsail.SetIpAddressTypeInput{
			IpAddressType: aws.String("dualstack"),
			ResourceName:  aws.String(n.name),
		}); err != nil {
			log.Error(err)
		}

		// update node IP
		inst, err := n.svc.GetInstance(&lightsail.GetInstanceInput{InstanceName: aws.String(n.name)})
		if err != nil {
			log.Error(err)
			return
		}
		n.ip = aws.StringValue(inst.Instance.Ipv6Addresses[0])
	}

	// Update domain record
	n.ddnsClient.AddUpdateDomainRecords(n.network, n.ip)
}
