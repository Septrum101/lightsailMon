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
	"github.com/sirupsen/logrus"

	"github.com/thank243/lightsailMon/common/ddns"
	"github.com/thank243/lightsailMon/common/notify"
	"github.com/thank243/lightsailMon/config"
)

func New(configNode *config.Node) *Node {
	n := new(Node)
	n.logger = logrus.WithFields(map[string]interface{}{
		"domain": configNode.Domain,
	})

	// create account session
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(
			configNode.AccessKeyID,
			configNode.SecretAccessKey,
			"",
		),
	})
	if err != nil {
		n.logger.Panic(err)
	}

	// init node
	n.svc = lightsail.New(sess, aws.NewConfig().WithRegion(configNode.Region))
	n.name = configNode.InstanceName
	n.network = configNode.Network
	n.domain = configNode.Domain
	n.port = configNode.Port
	n.timeout = time.Second * 5

	// Get lightsail instance IP and sync to domain
	inst, err := n.svc.GetInstance(&lightsail.GetInstanceInput{InstanceName: aws.String(n.name)})
	if err != nil {
		n.logger.Error(err)
	} else {
		switch n.network {
		case "tcp4":
			n.ip = aws.StringValue(inst.Instance.PublicIpAddress)
		case "tcp6":
			n.ip = aws.StringValue(inst.Instance.Ipv6Addresses[0])
		}
	}

	return n
}

// attachIP is a helper function to attach static IP to instance
func (n *Node) attachIP() {
	n.logger.Debug("Attach static IP")
	if _, err := n.svc.AttachStaticIp(&lightsail.AttachStaticIpInput{
		InstanceName: aws.String(n.name),
		StaticIpName: aws.String("LightsailMon"),
	}); err != nil {
		n.logger.Error(err)
	}
}

// detachIP is a helper function to detach static IP from instance
func (n *Node) detachIP() {
	n.logger.Debug("Detach static IP")
	if _, err := n.svc.DetachStaticIp(&lightsail.DetachStaticIpInput{
		StaticIpName: aws.String("LightsailMon"),
	}); err != nil {
		n.logger.Error(err)
	}
}

// disableDualStack is a helper function to disable dual stack network
func (n *Node) disableDualStack() {
	n.logger.Debug("Disable dual-stack network")
	if _, err := n.svc.SetIpAddressTypeRequest(&lightsail.SetIpAddressTypeInput{
		IpAddressType: aws.String("ipv4"),
		ResourceName:  aws.String(n.name),
	}); err != nil {
		n.logger.Error(err)
	}
}

// enableDualStack is a helper function to enable dual stack network
func (n *Node) enableDualStack() {
	n.logger.Debug("Enable dual-stack network")
	if _, err := n.svc.SetIpAddressTypeRequest(&lightsail.SetIpAddressTypeInput{
		IpAddressType: aws.String("dualstack"),
		ResourceName:  aws.String(n.name),
	}); err != nil {
		n.logger.Error(err)
	}
}

// setNodeIP is a helper function to update instance IP address
func (n *Node) setNodeIP(ipType string) {
	inst, err := n.svc.GetInstance(&lightsail.GetInstanceInput{InstanceName: aws.String(n.name)})
	if err != nil {
		n.logger.Error(err)
	}

	switch ipType {
	case "ipv4":
		n.ip = aws.StringValue(inst.Instance.PublicIpAddress)
	case "ipv6":
		n.ip = aws.StringValue(inst.Instance.Ipv6Addresses[0])
	}
}

func (n *Node) RenewIP() {
	n.logger.Warn("Change node IP")
	isDone := false
	for i := 0; i < 3; i++ {
		switch n.network {
		case "tcp4":
			n.attachIP()
			time.Sleep(time.Second * 3)
			n.detachIP()
			n.setNodeIP("ipv4")
		case "tcp6":
			n.disableDualStack()
			time.Sleep(time.Second * 3)
			n.enableDualStack()
			n.setNodeIP("ipv6")
		}

		// check again connection
		if _, err := n.checkConnection(); err != nil {
			n.logger.Errorf("Renew IP post check: %v attempt retry.. (%d/3)", err, i+1)
		} else {
			n.logger.Info("Renew IP post check: success")
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
			if err := n.ddnsClient.AddUpdateDomainRecords(n.network, n.domain, n.ip); err != nil {
				n.logger.Error(err)
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
			if err := n.notifier.Webhook(n.domain, fmt.Sprintf("IP changed: %s", n.ip)); err != nil {
				n.logger.Error(err)
			} else {
				n.logger.Info("Push message success")
			}
		} else {
			if err := n.notifier.Webhook(n.domain, fmt.Sprintf("[%s] Connection block after IP refresh 3 times", n.domain)); err != nil {
				n.logger.Error(err)
			} else {
				n.logger.Infof("Push message success")
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
			n.logger.Debugf("%v attempt retry.. (%d/3)", err, i+1)
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
			domainIps, err = n.ddnsClient.GetDomainRecords("A", n.domain)
		case "tcp6":
			domainIps, err = n.ddnsClient.GetDomainRecords("AAAA", n.domain)
		}
		if err != nil {
			n.logger.Error(err)
			return
		}

		if _, ok := domainIps[n.ip]; !ok {
			if err := n.ddnsClient.AddUpdateDomainRecords(n.network, n.domain, n.ip); err != nil {
				n.logger.Error(err)
			}
		}
	}
}

func (n *Node) IsBlock() bool {
	if delay, err := n.checkConnection(); err != nil {
		var v *net.OpError
		if errors.As(err, &v) && v.Addr != nil {
			n.logger.Error(err)
			return true
		}
	} else {
		n.logger.Infof("Tcping: %d ms", delay)
	}
	return false
}

func (n *Node) SetTimeout(t int) {
	n.timeout = time.Second * time.Duration(t)
}

func (n *Node) SetNotifier(notify notify.Notify) {
	n.notifier = notify
}

func (n *Node) SetDDNSClient(cli ddns.Client) {
	n.ddnsClient = cli
}

func (n *Node) Domain() string {
	return n.domain
}

func (n *Node) GetSvc() *lightsail.Lightsail {
	return n.svc
}
