package controller

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/go-resty/resty/v2"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	"github.com/Septrum101/lightsailMon/app/node"
	"github.com/Septrum101/lightsailMon/config"
)

func New(c *config.Config) *Service {
	s := &Service{
		conf:     c,
		cron:     cron.New(),
		internal: c.Internal,
		timeout:  c.Timeout,
		worker:   make(chan bool, c.Concurrent),
	}

	// init log level
	if l, err := log.ParseLevel(c.LogLevel); err != nil {
		log.Panic(err)
	} else {
		log.SetLevel(l)
		if l == log.DebugLevel {
			log.SetReportCaller(true)
		}
	}

	isNotify := c.Notify != nil && c.Notify.Enable
	isDDNS := c.DDNS != nil && c.DDNS.Enable

	ddnsStatus := "off"
	if isDDNS {
		ddnsStatus = strings.Title(c.DDNS.Provider)
	}
	notifierStatus := "off"
	if isNotify {
		notifierStatus = strings.Title(c.Notify.Provider)
	}
	fmt.Printf("Log level: %s, Concurrent: %d, DDNS: %s, Notifier: %s\n", c.LogLevel, c.Concurrent,
		ddnsStatus, notifierStatus)

	nodes := s.buildNodes(isNotify, isDDNS)
	if len(nodes) == 0 {
		log.Panic("no valid node")
	}
	s.nodes = nodes

	return s
}

func (s *Service) Start() {
	s.running = true

	// On init start, do once check
	log.Info("Initial connection test..")
	s.Run()

	// cron check
	if _, err := s.cron.AddJob(fmt.Sprintf("@every %ds", s.internal),
		cron.NewChain(cron.SkipIfStillRunning(cron.DefaultLogger)).Then(s)); err != nil {
		log.Panic(err)
	}

	s.cron.Start()
	log.Warnln(config.AppName, "Started")
}

func (s *Service) Close() {
	log.Infoln(config.AppName, "Closing..")
	entry := s.cron.Entries()
	for i := range entry {
		s.cron.Remove(entry[i].ID)
	}
	s.cron.Stop()
	close(s.worker)
	s.running = false
}

func (s *Service) Run() {
	// check local network connectivity
	if !checkIpv4() {
		return
	}

	if checkIpv6() {
		s.isIpv6 = true
	}

	s.changeNodeIps(s.getBlockNodes())
}

func checkIpv4() bool {
	start := time.Now()
	resp, err := resty.New().SetRetryCount(3).R().Get("http://detectportal.firefox.com/success.txt")
	if err != nil {
		log.Error(err)
		return false
	}
	delay := time.Since(start)
	if resp.StatusCode() > 299 {
		log.Error(resp.String())
		return false
	}
	log.WithField("domian", "ipv4.connectivity").Infof("Tcping: %d ms", delay.Milliseconds())
	return true
}

func checkIpv6() bool {
	dailer := new(net.Dialer)
	httpTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dailer.DialContext(ctx, "tcp6", addr)
		},
	}

	start := time.Now()
	resp, err := resty.New().SetTransport(httpTransport).SetRetryCount(3).R().Get("http://detectportal.firefox.com/success.txt")
	if err != nil {
		log.Error(err)
		return false
	}
	delay := time.Since(start)
	if resp.StatusCode() > 299 {
		log.Error(resp.String())
		return false
	}
	log.WithField("domian", "ipv6.connectivity").Infof("Tcping: %d ms", delay.Milliseconds())
	return true
}

func (s *Service) changeNodeIps(blockNodes []*node.Node) {
	if len(blockNodes) > 0 {
		// get blocked node lightsail service
		svcMap := make(map[*lightsail.Lightsail]bool)
		for _, node := range blockNodes {
			svcMap[node.Svc] = true
		}

		s.allocateStaticIps(svcMap)

		// handle change block IP
		for i := range blockNodes {
			s.worker <- true
			s.wg.Add(1)

			go func(n *node.Node) {
				defer func() {
					s.wg.Done()
					<-s.worker
				}()

				n.RenewIP()
			}(blockNodes[i])
		}
		s.wg.Wait()

		// release static IPs
		for svc := range svcMap {
			s.releaseStaticIps(svc)
		}
	}
}

// Release and Allocate Static Ip
func (s *Service) allocateStaticIps(svcMap map[*lightsail.Lightsail]bool) {
	for svc := range svcMap {
		s.releaseStaticIps(svc)

		log.WithField("region", *svc.Config.Region).Debug("Allocate region static IP")
		if _, err := svc.AllocateStaticIp(&lightsail.AllocateStaticIpInput{
			StaticIpName: aws.String("LightsailMon"),
		}); err != nil {
			log.Error(err)
		}
	}
}

func (s *Service) getBlockNodes() []*node.Node {
	nodesChan := make(chan *node.Node)

	// get block nodes
	for i := range s.nodes {
		s.worker <- true
		s.wg.Add(1)

		go func(i int) {
			defer func() {
				<-s.worker
				s.wg.Done()
			}()

			n := s.nodes[i]

			// check host ipv6 is availiable
			if n.Network == "tcp4" && !s.isIpv6 {
				log.Error("Host's ipv6 network is not supported")
				return
			}

			go func() {
				if err := n.UpdateDomainIp(); err != nil {
					n.Logger.Errorf("Failed to update domain IP: %v", err)
				}
			}()

			if n.IsBlock() {
				// add to blockNodes channel
				nodesChan <- n
			}
		}(i)
	}

	// wait after all node is checked
	go func() {
		s.wg.Wait()
		close(nodesChan)
	}()

	// read blocked nodes from channel
	var blockedNodes []*node.Node
	for n := range nodesChan {
		blockedNodes = append(blockedNodes, n)
	}

	return blockedNodes
}

func (s *Service) releaseStaticIps(svc *lightsail.Lightsail) {
	log.WithField("region", *svc.Config.Region).Debug("Release region static IPs")
	if ips, err := svc.GetStaticIps(&lightsail.GetStaticIpsInput{}); err != nil {
		log.Error(err)
	} else {
		for i := range ips.StaticIps {
			ip := ips.StaticIps[i]
			if _, err := svc.ReleaseStaticIp(&lightsail.ReleaseStaticIpInput{StaticIpName: ip.Name}); err != nil {
				log.Error(err)
			}
		}
	}
}
