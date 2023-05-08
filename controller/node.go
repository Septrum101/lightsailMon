package controller

import (
	"fmt"
	"net"
	"strconv"
	"time"

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
	if err := n.ddnsClient.AddUpdateDomainRecords(n.network, n.ip); err != nil {
		log.Error(err)
		return
	}

	if err := n.notifier.Webhook(fmt.Sprintf("[%s] IP changed: %s", n.domain, n.ip)); err != nil {
		log.Error(err)
	} else {
		log.Infof("[%s:%d] Push message success", n.domain, n.port)
	}
}

func (n *node) checkConnection(t int) (int64, error) {
	var (
		conn    net.Conn
		err     error
		ip      = n.ip
		port    = n.port
		network = n.network
	)

	for i := 0; ; i++ {
		if network == "tcp6" {
			ip = "[" + ip + "]"
		}
		start := time.Now()
		conn, err = net.DialTimeout(network, ip+":"+strconv.Itoa(port), time.Second*time.Duration(t))
		d := time.Since(start)
		if err != nil {
			log.Infof("%v attempt retry.. (%d/3)", err, i+1)
			if i == 2 {
				return 0, err
			}
			time.Sleep(time.Second * 5)
		} else {
			conn.Close()
			return d.Milliseconds(), nil
		}
	}
}
