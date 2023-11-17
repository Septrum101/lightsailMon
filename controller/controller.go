package controller

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/go-resty/resty/v2"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	"github.com/thank243/lightsailMon/app/node"
	"github.com/thank243/lightsailMon/config"
)

func New(c *config.Config) *Service {
	s := &Service{
		conf:     c,
		cron:     cron.New(),
		internal: c.Internal,
		timeout:  c.Timeout,
		worker:   make(chan uint8, c.Concurrent),
	}

	// init log level
	if l, err := log.ParseLevel(c.LogLevel); err != nil {
		log.Panic(err)
	} else {
		log.SetLevel(l)
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
	fmt.Printf("Log level: %s  (Concurrent: %d, DDNS: %s, Notifier: %s)\n", c.LogLevel, c.Concurrent,
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
	// check local network connection
	resp, err := resty.New().SetRetryCount(3).R().Get("http://connectivitycheck.platform.hicloud.com/generate_204")
	if err != nil {
		log.Error(err)
		return
	}
	if resp.StatusCode() != 204 {
		log.Error(resp.String())
		return
	}

	s.handler()
}

func (s *Service) handler() {
	var blockNodes []*node.Node
	svcMap := make(map[*lightsail.Lightsail]bool)

	// get block nodes
	for i := range s.nodes {
		s.workerAdd()

		go func(n *node.Node) {
			defer s.workerDone()

			go n.UpdateDomainIp()
			if n.IsBlock() {
				s.Lock()
				defer s.Unlock()
				// add to blockNodes
				blockNodes = append(blockNodes, n)
				svcMap[n.GetSvc()] = true
			}
		}(s.nodes[i])
	}
	s.wg.Wait()

	// change block nodes ip
	if len(blockNodes) > 0 {
		// Release and Allocate Static Ip
		for svc := range svcMap {
			s.releaseStaticIps(svc)

			log.Debugf("[Region: %s] Allocate region static IP", *svc.Config.Region)
			if _, err := svc.AllocateStaticIp(&lightsail.AllocateStaticIpInput{
				StaticIpName: aws.String("LightsailMon"),
			}); err != nil {
				log.Error(err)
			}
		}

		// handle change block IP
		for i := range blockNodes {
			s.workerAdd()

			go func(n *node.Node) {
				defer s.workerDone()
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

func (s *Service) workerAdd() {
	s.wg.Add(1)
	s.worker <- 0
}

func (s *Service) workerDone() {
	<-s.worker
	s.wg.Done()
}

func (s *Service) releaseStaticIps(svc *lightsail.Lightsail) {
	log.Debugf("[Region: %s] Release region static IPs", *svc.Config.Region)
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
