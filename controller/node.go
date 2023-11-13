package controller

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lightsail"
	log "github.com/sirupsen/logrus"

	"github.com/thank243/lightsailMon/common/ddns/cloudflare"
	"github.com/thank243/lightsailMon/common/ddns/google"
	"github.com/thank243/lightsailMon/common/notify"
	"github.com/thank243/lightsailMon/config"
	"github.com/thank243/lightsailMon/utils"
)

func buildNodes(c *config.Config) (nodes []*node) {
	for i := range c.Nodes {
		n := c.Nodes[i]
		// create account session
		sess, err := session.NewSession(&aws.Config{
			Credentials: credentials.NewStaticCredentials(
				n.AccessKeyID,
				n.SecretAccessKey,
				"",
			),
		})
		if err != nil {
			log.Panic(err)
		}

		// init node
		newNode := &node{
			name:    n.InstanceName,
			network: n.Network,
			domain:  n.Domain,
			port:    n.Port,
			svc:     lightsail.New(sess, aws.NewConfig().WithRegion(n.Region)),
		}

		// init ddns client
		if c.DDNS != nil && c.DDNS.Enable {
			switch c.DDNS.Provider {
			case "cloudflare":
				if newNode.ddnsClient, err = cloudflare.New(c.DDNS.Config, n.Domain); err != nil {
					log.Panicln(err)
				}
			case "google":
				if newNode.ddnsClient, err = google.New(c.DDNS.Config, n.Domain); err != nil {
					log.Panicln(err)
				}
			}
		}

		// Get lightsail instance IP and sync to domain
		inst, err := newNode.svc.GetInstance(&lightsail.GetInstanceInput{InstanceName: aws.String(newNode.name)})
		if err != nil {
			log.Error(err)
		} else {
			switch newNode.network {
			case "tcp4":
				newNode.ip = aws.StringValue(inst.Instance.PublicIpAddress)
			case "tcp6":
				newNode.ip = aws.StringValue(inst.Instance.Ipv6Addresses[0])
			}
			if newNode.ip != utils.GetDomainIP(newNode.network, newNode.domain) {
				log.Infof("[%s] sync node ip to domain", n.Domain)
				log.Error(newNode.ddnsClient.AddUpdateDomainRecords(newNode.network, newNode.ip))
			}
		}

		// init notifier
		if c.Notify != nil && c.Notify.Enable {
			switch c.Notify.Provider {
			case "pushplus":
				newNode.notifier = &notify.PushPlus{Token: c.Notify.Config["pushplus_token"].(string)}
			case "telegram":
				newNode.notifier = &notify.Telegram{
					ChatID: int64(c.Notify.Config["telegram_chatid"].(int)),
					Token:  c.Notify.Config["telegram_token"].(string),
				}
			}
		}
		nodes = append(nodes, newNode)
	}

	return nodes
}

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

			time.Sleep(time.Second * 5)

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

			time.Sleep(time.Second * 5)

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
		for i := 0; i < 3; i++ {
			if err := n.ddnsClient.AddUpdateDomainRecords(n.network, n.ip); err != nil {
				log.Error(err)
				time.Sleep(time.Second * 5)
			} else {
				break
			}
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
