package node

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lightsail"
	log "github.com/sirupsen/logrus"

	"github.com/thank243/lightsailMon/common/ddns"
	"github.com/thank243/lightsailMon/common/notify"
	"github.com/thank243/lightsailMon/config"
	"github.com/thank243/lightsailMon/helper"
)

func New(configNode *config.Node) *Node {
	// create account session
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(
			configNode.AccessKeyID,
			configNode.SecretAccessKey,
			"",
		),
	})
	if err != nil {
		log.Panic(err)
	}

	// init node
	node := &Node{
		svc:     lightsail.New(sess, aws.NewConfig().WithRegion(configNode.Region)),
		name:    configNode.InstanceName,
		network: configNode.Network,
		domain:  configNode.Domain,
		port:    configNode.Port,
		timeout: time.Second * 5,
	}

	// Get lightsail instance IP and sync to domain
	inst, err := node.svc.GetInstance(&lightsail.GetInstanceInput{InstanceName: aws.String(node.name)})
	if err != nil {
		log.Error(err)
	} else {
		switch node.network {
		case "tcp4":
			node.ip = aws.StringValue(inst.Instance.PublicIpAddress)
		case "tcp6":
			node.ip = aws.StringValue(inst.Instance.Ipv6Addresses[0])
		}
		if node.ip != helper.GetDomainIP(node.network, node.domain) {
			log.Infof("[%s] Sync node ip with domain", node.domain)
			err := node.ddnsClient.AddUpdateDomainRecords(node.network, node.ip)
			if err != nil {
				log.Error(err)
			}
		}
	}

	return node
}

// attachIP is a helper function to attach static IP to instance
func (n *Node) attachIP() {
	log.Debugf("[%s] Attach static IP", n.domain)
	if _, err := n.svc.AttachStaticIp(&lightsail.AttachStaticIpInput{
		InstanceName: aws.String(n.name),
		StaticIpName: aws.String("LightsailMon"),
	}); err != nil {
		log.Error(err)
	}
}

// detachIP is a helper function to detach static IP from instance
func (n *Node) detachIP() {
	log.Debugf("[%s] Detach static IP", n.domain)
	if _, err := n.svc.DetachStaticIp(&lightsail.DetachStaticIpInput{
		StaticIpName: aws.String("LightsailMon"),
	}); err != nil {
		log.Error(err)
	}
}

// disableDualStack is a helper function to disable dual stack network
func (n *Node) disableDualStack() {
	log.Debugf("[%s] Disable dual-stack network", n.domain)
	if _, err := n.svc.SetIpAddressTypeRequest(&lightsail.SetIpAddressTypeInput{
		IpAddressType: aws.String("ipv4"),
		ResourceName:  aws.String(n.name),
	}); err != nil {
		log.Error(err)
	}
}

// enableDualStack is a helper function to enable dual stack network
func (n *Node) enableDualStack() {
	log.Debugf("[%s] Enable dual-stack network", n.domain)
	if _, err := n.svc.SetIpAddressTypeRequest(&lightsail.SetIpAddressTypeInput{
		IpAddressType: aws.String("dualstack"),
		ResourceName:  aws.String(n.name),
	}); err != nil {
		log.Error(err)
	}
}

// setNodeIP is a helper function to update instance IP address
func (n *Node) setNodeIP(ipType string) {
	inst, err := n.svc.GetInstance(&lightsail.GetInstanceInput{InstanceName: aws.String(n.name)})
	if err != nil {
		log.Error(err)
	}

	switch ipType {
	case "ipv4":
		n.ip = aws.StringValue(inst.Instance.PublicIpAddress)
	case "ipv6":
		n.ip = aws.StringValue(inst.Instance.Ipv6Addresses[0])
	}
}

func (n *Node) RenewIP() {
	log.Warnf("[%s] Change node IP", n.domain)
	isDone := false
	for i := 0; i < 3; i++ {
		switch n.network {
		case "tcp4":
			n.attachIP()
			time.Sleep(time.Second * 5)
			n.detachIP()
			n.setNodeIP("ipv4")
		case "tcp6":
			n.disableDualStack()
			time.Sleep(time.Second * 5)
			n.enableDualStack()
			n.setNodeIP("ipv6")
		}

		// check again connection
		if _, err := n.checkConnection(); err != nil {
			log.Errorf("[%s] Renew IP post check: %v attempt retry.. (%d/3)", n.domain, err, i+1)
		} else {
			log.Infof("[%s] Renew IP post check: success", n.domain)
			isDone = true
			break
		}
	}

	n.pushMessage(isDone)
	n.updateDomain()
}

// Update domain record
func (n *Node) updateDomain() {
	if n.ddnsClient != nil {
		for i := 0; i < 3; i++ {
			if err := n.ddnsClient.AddUpdateDomainRecords(n.network, n.ip); err != nil {
				log.Error(err)
				if i == 2 {
					return
				}
				time.Sleep(time.Second * 5)
			} else {
				return
			}
		}
	}
}

// push message
func (n *Node) pushMessage(isDone bool) {
	if n.notifier != nil {
		if isDone {
			if err := n.notifier.Webhook(n.domain, fmt.Sprintf("[%s] IP changed: %s", n.domain, n.ip)); err != nil {
				log.Error(err)
			} else {
				log.Infof("[%s:%d] Push message success", n.domain, n.port)
			}
		} else {
			if err := n.notifier.Webhook(n.domain, fmt.Sprintf("[%s] Connection block after IP refresh 3 times", n.domain)); err != nil {
				log.Error(err)
			} else {
				log.Infof("[%s] Push message success", n.domain)
			}
		}
	}
}

func (n *Node) checkConnection() (int64, error) {
	var (
		conn net.Conn
		err  error
	)

	if n.network == "tcp6" {
		n.ip = "[" + n.ip + "]"
	}

	for i := 0; i < 3; i++ {
		start := time.Now()
		conn, err = net.DialTimeout(n.network, n.ip+":"+strconv.Itoa(n.port), n.timeout)
		if err != nil {
			log.Debugf("%v attempt retry.. (%d/3)", err, i+1)
		} else {
			conn.Close()
			return time.Since(start).Milliseconds(), nil
		}
		time.Sleep(time.Second * 5)
	}

	return 0, err
}

func (n *Node) UpdateDomainIp() {
	// check domain sync with ip
	if n.ddnsClient != nil {
		var (
			domainIps map[string]bool
			err       error
		)
		switch n.network {
		case "tcp4":
			domainIps, err = n.ddnsClient.GetDomainRecords("A")
		case "tcp6":
			domainIps, err = n.ddnsClient.GetDomainRecords("AAAA")
		}
		if err != nil {
			log.Error(err)
			return
		}

		if _, ok := domainIps[n.ip]; !ok {
			if err := n.ddnsClient.AddUpdateDomainRecords(n.network, n.ip); err != nil {
				log.Error(err)
			}
		}
	}
}

func (n *Node) IsBlock() bool {
	addr := fmt.Sprintf("%s:%d", n.domain, n.port)
	credValue, _ := n.svc.Config.Credentials.Get()

	if delay, err := n.checkConnection(); err != nil {
		var v *net.OpError
		if errors.As(err, &v) && v.Addr != nil {
			log.Errorf("[AKID: %s] %s %v", credValue.AccessKeyID, addr, err)
			return true
		}
	} else {
		log.Infof("[%s] Tcping: %d ms", addr, delay)
	}
	return false
}

func (n *Node) SetTimeout(t int) {
	n.timeout = time.Second * time.Duration(t)
}

func (n *Node) SetNotifier(notify notify.Notify) {
	n.notifier = notify
}

func (n *Node) SetDdnsClient(cli ddns.Client) {
	n.ddnsClient = cli
}

func (n *Node) Domain() string {
	return n.domain
}

func (n *Node) GetSvc() *lightsail.Lightsail {
	return n.svc
}
