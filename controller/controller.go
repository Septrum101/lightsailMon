package controller

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/go-resty/resty/v2"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	"github.com/thank243/lightsailMon/config"
)

func New(c *config.Config) *Server {
	s := &Server{
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

	ddnsStatus := "off"
	if c.DDNS.Enable {
		ddnsStatus = strings.Title(c.DDNS.Provider)
	}
	notifierStatus := "off"
	if c.Notify.Enable {
		notifierStatus = strings.Title(c.Notify.Provider)
	}
	fmt.Printf("Log level: %s  (Concurrent: %d, DDNS: %s, Notifier: %s)\n", c.LogLevel, c.Concurrent,
		ddnsStatus, notifierStatus)

	s.nodes = buildNodes(c)

	return s
}

func (s *Server) Start() {
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

func (s *Server) Run() {
	// check local network connection
	resp, err := resty.New().SetRetryCount(3).R().Get("http://www.gstatic.com/generate_204")
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

func (s *Server) handler() {
	var blockNodes []*node
	svcMap := make(map[*lightsail.Lightsail]uint8)

	for k := range s.nodes {
		s.wg.Add(1)
		s.worker <- 0

		go func(n *node) {
			defer func() {
				<-s.worker
				s.wg.Done()
			}()

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
				} else {
					if _, ok := domainIps[n.ip]; !ok {
						if err := n.ddnsClient.AddUpdateDomainRecords(n.network, n.ip); err != nil {
							log.Error(err)
						}
					}
				}
			}

			// check connection
			addr := fmt.Sprint(n.domain + ":" + strconv.Itoa(n.port))
			credValue, _ := n.svc.Config.Credentials.Get()

			if delay, err := n.checkConnection(s.timeout); err != nil {
				if v, ok := err.(*net.OpError); ok && v.Addr != nil {
					s.Lock()
					defer s.Unlock()
					// add to blockNodes
					blockNodes = append(blockNodes, n)
					svcMap[n.svc] = 0
				}
				log.Errorf("[AKID: %s] %s %v", credValue.AccessKeyID, addr, err)
			} else {
				log.Infof("[%s] Tcping: %d ms", addr, delay)
			}

		}(s.nodes[k])
	}
	s.wg.Wait()

	if len(blockNodes) > 0 {
		// Release and Allocate Static Ip
		for svc := range svcMap {
			log.Debugf("[Region: %s] Release region static IP", *svc.Config.Region)
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

			log.Debugf("[Region: %s] Allocate region static IP", *svc.Config.Region)
			if _, err := svc.AllocateStaticIp(&lightsail.AllocateStaticIpInput{
				StaticIpName: aws.String("LightsailMon"),
			}); err != nil {
				log.Error(err)
			}
		}

		// handle change block IP
		for i := range blockNodes {
			s.wg.Add(1)
			s.worker <- 0

			go func(n *node) {
				defer func() {
					<-s.worker
					s.wg.Done()
				}()

				log.Errorf("[%s:%d] Change node IP", n.domain, n.port)

				n.renewIP()
			}(blockNodes[i])
		}
		s.wg.Wait()

		// release static IPs
		for svc := range svcMap {
			log.Debugf("[Region: %s] Release region static IP", *svc.Config.Region)
			if _, err := svc.ReleaseStaticIp(&lightsail.ReleaseStaticIpInput{
				StaticIpName: aws.String("LightsailMon"),
			}); err != nil {
				log.Error(err)
			}
		}
	}
}

func (s *Server) Close() {
	log.Infoln(config.AppName, "Closing..")
	entry := s.cron.Entries()
	for i := range entry {
		s.cron.Remove(entry[i].ID)
	}
	s.cron.Stop()
	close(s.worker)
	s.running = false
}
