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

func (n *node) renewIP() {
	isDone := false
	for i := 0; i < 3; i++ {
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
				continue
			}
			n.ip = aws.StringValue(inst.Instance.PublicIpAddress)
		case "tcp6":
			// disable dual-stack network
			log.Debugf("[%s:%d] Disable dual-stack network", n.domain, n.port)
			if _, err := n.svc.SetIpAddressTypeRequest(&lightsail.SetIpAddressTypeInput{
				IpAddressType: aws.String("ipv4"),
				ResourceName:  aws.String(n.name),
			}); err != nil {
				log.Error(err)
			}

			// enable dual-stack network
			log.Debugf("[%s:%d] Enable dual-stack network", n.domain, n.port)
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
				continue
			}
			n.ip = aws.StringValue(inst.Instance.Ipv6Addresses[0])
		}

		// check again connection
		if _, err := n.checkConnection(5); err != nil {
			log.Errorf("renew IP post check: %v attempt retry.. (%d/3)", err, i+1)
		} else {
			isDone = true
			log.Infof("renew IP post check: success")
			break
		}
	}

	// push message
	if n.notifier != nil {
		if isDone {
			if err := n.notifier.Webhook(n.domain, fmt.Sprintf("IP changed: %s", n.ip)); err != nil {
				log.Error(err)
			} else {
				log.Infof("[%s:%d] Push message success", n.domain, n.port)
			}
		} else {
			if err := n.notifier.Webhook(n.domain, "Connection block after IP refresh 3 times"); err != nil {
				log.Error(err)
			} else {
				log.Infof("[%s:%d] Push message success", n.domain, n.port)
			}
		}
	}

	// Update domain record
	if n.ddnsClient != nil {
		if err := n.ddnsClient.AddUpdateDomainRecords(n.network, n.ip); err != nil {
			log.Error(err)
		}
	}
}

func (n *node) checkConnection(t int) (int64, error) {
	var (
		conn net.Conn
		err  error
	)

	if n.network == "tcp6" {
		n.ip = "[" + n.ip + "]"
	}

	for i := 0; i < 3; i++ {
		if i > 0 {
			time.Sleep(time.Second * 5)
		}

		start := time.Now()
		conn, err = net.DialTimeout(n.network, n.ip+":"+strconv.Itoa(n.port), time.Second*time.Duration(t))
		if err != nil {
			log.Debugf("%v attempt retry.. (%d/3)", err, i+1)
		} else {
			conn.Close()
			return time.Since(start).Milliseconds(), nil
		}
	}

	return 0, err
}
