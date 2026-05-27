package node

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lightsail"
	"github.com/aws/aws-sdk-go-v2/service/lightsail/types"
	"github.com/sirupsen/logrus"

	cfg "github.com/Septrum101/lightsailMon/config"
)

func New(configNode *cfg.Node) []*Node {
	var nodes []*Node
	for i := range configNode.Network {
		network := configNode.Network[i]
		n := &Node{
			Timeout: time.Second * 5,
			Logger: logrus.WithFields(map[string]interface{}{
				"domain": fmt.Sprintf("%s(%s)", configNode.Domain, network),
			}),
			name:    configNode.InstanceName,
			Network: network,
			port:    configNode.Port,
			domain:  configNode.Domain,
		}

		// create account session
		cfg, err := config.LoadDefaultConfig(context.Background(),
			config.WithRegion(configNode.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				configNode.AccessKeyID,
				configNode.SecretAccessKey,
				"",
			)),
		)
		if err != nil {
			n.Logger.Panic(err)
		}
		n.Svc = lightsail.NewFromConfig(cfg)

		// Get lightsail instance IP and sync to domain
		inst, err := n.Svc.GetInstance(context.Background(), &lightsail.GetInstanceInput{InstanceName: aws.String(n.name)})
		if err != nil {
			n.Logger.Error(err)
		} else {
			switch n.Network {
			case "tcp4":
				n.ip = aws.ToString(inst.Instance.PublicIpAddress)
			case "tcp6":
				if len(inst.Instance.Ipv6Addresses) > 0 {
					n.ip = aws.ToString(&inst.Instance.Ipv6Addresses[0])
				}
			}
		}

		nodes = append(nodes, n)
	}

	return nodes
}

// attachIP is a helper function to attach static IP to instance
func (n *Node) attachIP() {
	n.Logger.Debug("Attach static IP")
	if _, err := n.Svc.AttachStaticIp(context.Background(), &lightsail.AttachStaticIpInput{
		InstanceName: aws.String(n.name),
		StaticIpName: aws.String("LightsailMon"),
	}); err != nil {
		n.Logger.Error(err)
	}
}

// detachIP is a helper function to detach static IP from instance
func (n *Node) detachIP() {
	n.Logger.Debug("Detach static IP")
	if _, err := n.Svc.DetachStaticIp(context.Background(), &lightsail.DetachStaticIpInput{
		StaticIpName: aws.String("LightsailMon"),
	}); err != nil {
		n.Logger.Error(err)
	}
}

// disableDualStack is a helper function to disable dual stack network
func (n *Node) disableDualStack() {
	n.Logger.Debug("Disable dual-stack network")
	if _, err := n.Svc.SetIpAddressType(context.Background(), &lightsail.SetIpAddressTypeInput{
		IpAddressType: types.IpAddressTypeIpv4,
		ResourceName:  aws.String(n.name),
		ResourceType:  types.ResourceTypeInstance,
	}); err != nil {
		n.Logger.Error(err)
	}
}

// enableDualStack is a helper function to enable dual stack network
func (n *Node) enableDualStack() {
	n.Logger.Debug("Enable dual-stack network")
	if _, err := n.Svc.SetIpAddressType(context.Background(), &lightsail.SetIpAddressTypeInput{
		IpAddressType: types.IpAddressTypeDualstack,
		ResourceName:  aws.String(n.name),
		ResourceType:  types.ResourceTypeInstance,
	}); err != nil {
		n.Logger.Error(err)
	}
}

// setIp is a helper function to update instance IP address
func (n *Node) setIp(ipType string) {
	inst, err := n.Svc.GetInstance(context.Background(), &lightsail.GetInstanceInput{InstanceName: aws.String(n.name)})
	if err != nil {
		n.Logger.Error(err)
		return
	}

	switch ipType {
	case "ipv4":
		n.ip = aws.ToString(inst.Instance.PublicIpAddress)
	case "ipv6":
		n.ip = aws.ToString(&inst.Instance.Ipv6Addresses[0])
	}
}

func (n *Node) RenewIP() {
	n.Logger.Warn("Change node IP")
	isSuccess := false
	for i := 0; i < 3; i++ {
		switch n.Network {
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

	if err := n.updateDomain(); err != nil {
		n.Logger.Info(err)
	}

	if err := n.pushMessage(isSuccess); err != nil {
		n.Logger.Error(err)
	} else {
		n.Logger.Info("Push message success")
	}
}

// Update domain record
func (n *Node) updateDomain() error {
	if n.DdnsClient == nil {
		return errors.New("ddns client is null")
	}

	var err error
	for i := 0; i < 3; i++ {
		if err = n.DdnsClient.AddUpdateDomainRecords(n.Network, n.domain, n.ip); err != nil {
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
		if err := n.Notifier.Webhook(n.domain+"("+n.Network+")", fmt.Sprintf("IP changed: %s", n.ip)); err != nil {
			return err
		}
	} else {
		if err := n.Notifier.Webhook(n.domain+"("+n.Network+")", fmt.Sprintf("[%s] Connection block after IP refresh 3 times", n.domain)); err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) checkConnection() (int64, error) {
	addr := n.ip
	if n.Network == "tcp6" {
		addr = "[" + n.ip + "]"
	}

	d, conn, err := dialWithRetry(n.Network, addr+":"+strconv.Itoa(n.port), n.Timeout, 3, 5*time.Second)
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

	switch n.Network {
	case "tcp4":
		domainIps, err = n.DdnsClient.GetDomainRecords("A", n.domain)
	case "tcp6":
		domainIps, err = n.DdnsClient.GetDomainRecords("AAAA", n.domain)
	}
	if err != nil {
		return err
	}

	if _, ok := domainIps[n.ip]; !ok {
		if err := n.DdnsClient.AddUpdateDomainRecords(n.Network, n.domain, n.ip); err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) IsBlock() bool {
	delay, err := n.checkConnection()
	if err != nil {
		if pathErr, ok := errors.AsType[*net.OpError](err); ok && pathErr.Addr != nil {
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

	for range maxRetries {
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
