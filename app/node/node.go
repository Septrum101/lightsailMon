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

	"github.com/thank243/lightsailMon/config"
)

func New(configNode *config.Node) *Node {
	n := &Node{
		Timeout: time.Second * 5,
		Logger: logrus.WithFields(map[string]interface{}{
			"domain": configNode.Domain,
		}),
		name:    configNode.InstanceName,
		network: configNode.Network,
		port:    configNode.Port,
		domain:  configNode.Domain,
	}

	// create account session
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(
			configNode.AccessKeyID,
			configNode.SecretAccessKey,
			"",
		),
	})
	if err != nil {
		n.Logger.Panic(err)
	}
	n.Svc = lightsail.New(sess, aws.NewConfig().WithRegion(configNode.Region))

	// Get lightsail instance IP and sync to domain
	inst, err := n.Svc.GetInstance(&lightsail.GetInstanceInput{InstanceName: aws.String(n.name)})
	if err != nil {
		n.Logger.Error(err)
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
	n.Logger.Debug("Attach static IP")
	if _, err := n.Svc.AttachStaticIp(&lightsail.AttachStaticIpInput{
		InstanceName: aws.String(n.name),
		StaticIpName: aws.String("LightsailMon"),
	}); err != nil {
		n.Logger.Error(err)
	}
}

// detachIP is a helper function to detach static IP from instance
func (n *Node) detachIP() {
	n.Logger.Debug("Detach static IP")
	if _, err := n.Svc.DetachStaticIp(&lightsail.DetachStaticIpInput{
		StaticIpName: aws.String("LightsailMon"),
	}); err != nil {
		n.Logger.Error(err)
	}
}

// disableDualStack is a helper function to disable dual stack network
func (n *Node) disableDualStack() {
	n.Logger.Debug("Disable dual-stack network")
	if _, err := n.Svc.SetIpAddressTypeRequest(&lightsail.SetIpAddressTypeInput{
		IpAddressType: aws.String("ipv4"),
		ResourceName:  aws.String(n.name),
	}); err != nil {
		n.Logger.Error(err)
	}
}

// enableDualStack is a helper function to enable dual stack network
func (n *Node) enableDualStack() {
	n.Logger.Debug("Enable dual-stack network")
	if _, err := n.Svc.SetIpAddressTypeRequest(&lightsail.SetIpAddressTypeInput{
		IpAddressType: aws.String("dualstack"),
		ResourceName:  aws.String(n.name),
	}); err != nil {
		n.Logger.Error(err)
	}
}

// setIp is a helper function to update instance IP address
func (n *Node) setIp(ipType string) {
	inst, err := n.Svc.GetInstance(&lightsail.GetInstanceInput{InstanceName: aws.String(n.name)})
	if err != nil {
		n.Logger.Error(err)
	}

	switch ipType {
	case "ipv4":
		n.ip = aws.StringValue(inst.Instance.PublicIpAddress)
	case "ipv6":
		n.ip = aws.StringValue(inst.Instance.Ipv6Addresses[0])
	}
}

func (n *Node) RenewIP() {
	n.Logger.Warn("Change node IP")
	isSuccess := false
	for i := 0; i < 3; i++ {
		switch n.network {
		case "tcp4":
			n.attachIP()
			time.Sleep(time.Second * 3)
			n.detachIP()
			n.setIp("ipv4")
		case "tcp6":
			n.disableDualStack()
			time.Sleep(time.Second * 3)
			n.enableDualStack()
			n.setIp("ipv6")
		}

		// check again connection
		if _, err := n.checkConnection(); err != nil {
			n.Logger.Errorf("Renew IP post check: %v attempt retry.. (%d/3)", err, i+1)
		} else {
			n.Logger.Info("Renew IP post check: success")
			isSuccess = true
			break
		}
	}

	if err := n.pushMessage(isSuccess); err != nil {
		n.Logger.Error(err)
	} else {
		n.Logger.Info("Push message success")
	}

	if err := n.updateDomain(); err != nil {
		n.Logger.Info(err)
	}
}

// Update domain record
func (n *Node) updateDomain() error {
	if n.DdnsClient == nil {
		return errors.New("ddns client is null")
	}

	var err error
	for i := 0; i < 3; i++ {
		if err = n.DdnsClient.AddUpdateDomainRecords(n.network, n.domain, n.ip); err != nil {
			time.Sleep(time.Second * 5)
			continue
		}

		return nil
	}

	return err
}

// push message
func (n *Node) pushMessage(isSuccess bool) error {
	if n.Notifier == nil {
		return errors.New("notifier is null")
	}

	if isSuccess {
		if err := n.Notifier.Webhook(n.domain, fmt.Sprintf("IP changed: %s", n.ip)); err != nil {
			return err
		}
	} else {
		if err := n.Notifier.Webhook(n.domain, fmt.Sprintf("[%s] Connection block after IP refresh 3 times", n.domain)); err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) checkConnection() (int64, error) {
	addr := n.ip
	if n.network == "tcp6" {
		addr = "[" + n.ip + "]"
	}

	d, conn, err := dialWithRetry(n.network, addr+":"+strconv.Itoa(n.port), n.Timeout, 3, 5*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	return d, err
}

func (n *Node) UpdateDomainIp() error {
	if n.DdnsClient == nil {
		return errors.New("ddns client is null")
	}

	// check domain sync with ip
	var (
		domainIps map[string]bool
		err       error
	)

	switch n.network {
	case "tcp4":
		domainIps, err = n.DdnsClient.GetDomainRecords("A", n.domain)
	case "tcp6":
		domainIps, err = n.DdnsClient.GetDomainRecords("AAAA", n.domain)
	}
	if err != nil {
		return err
	}

	if _, ok := domainIps[n.ip]; !ok {
		if err := n.DdnsClient.AddUpdateDomainRecords(n.network, n.domain, n.ip); err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) IsBlock() bool {
	delay, err := n.checkConnection()
	if err != nil {
		var v *net.OpError
		if errors.As(err, &v) && v.Addr != nil {
			n.Logger.Errorf("after 3 attempts, last error: %s", err)
			return true
		}
		n.Logger.Errorf("after 3 attempts, last error: %s", err)
		return false
	}

	n.Logger.Infof("Tcping: %d ms", delay)
	return false
}

func dialWithRetry(network, address string, timeout time.Duration, maxRetries int, delay time.Duration) (int64, net.Conn, error) {
	var err error
	var conn net.Conn

	for i := 0; i < maxRetries; i++ {
		start := time.Now()
		conn, err = net.DialTimeout(network, address, timeout)
		if err == nil {
			return time.Since(start).Milliseconds(), conn, nil
		}

		// sleep for a while before trying again
		time.Sleep(delay)
	}

	return 0, nil, err
}
