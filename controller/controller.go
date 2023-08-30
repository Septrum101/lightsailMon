package controller

import (
	"fmt"
	"net"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/go-resty/resty/v2"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	"github.com/thank243/lightsailMon/common/ddns/cloudflare"
	"github.com/thank243/lightsailMon/common/ddns/google"
	"github.com/thank243/lightsailMon/common/notify"
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

	// init notifier
	var (
		noti       notify.Notify
		enableNoti bool
	)
	if c.Notify != nil && c.Notify.Enable {
		enableNoti = true
		switch c.Notify.Provider {
		case "pushplus":
			noti = &notify.PushPlus{Token: c.Notify.Config["pushplus_token"].(string)}
		case "telegram":
			noti = &notify.Telegram{
				ChatID: int64(c.Notify.Config["telegram_chatid"].(int)),
				Token:  c.Notify.Config["telegram_token"].(string),
			}
		}
	}

	ddnsStatus := "off"
	if c.DDNS.Enable {
		ddnsStatus = c.DDNS.Provider
	}
	notifierStatus := "off"
	if c.Notify.Enable {
		notifierStatus = c.Notify.Provider
	}
	fmt.Printf("Log level: %s  (Concurrent: %d, DDNS: %s, Notifier: %s)\n", c.LogLevel, c.Concurrent,
		ddnsStatus, notifierStatus)

	for i := range c.Accounts {
		a := c.Accounts[i]
		// create account session
		sess, err := session.NewSession(&aws.Config{
			Credentials: credentials.NewStaticCredentials(
				a.AccessKeyID,
				a.SecretAccessKey,
				"",
			),
		})
		if err != nil {
			log.Panic(err)
		}

		for ii := range a.Regions {
			r := a.Regions[ii]
			// init region svc
			svc := lightsail.New(sess, aws.NewConfig().WithRegion(r.Name))

			for iii := range r.Nodes {
				n := r.Nodes[iii]
				initNode := &node{
					name:    n.InstanceName,
					network: n.Network,
					domain:  n.Domain,
					port:    n.Port,
					svc:     svc,
				}
				// init ddns client
				if c.DDNS != nil && c.DDNS.Enable {
					switch c.DDNS.Provider {
					case "cloudflare":
						if initNode.ddnsClient, err = cloudflare.New(c.DDNS.Config, n.Domain); err != nil {
							log.Panicln(err)
						}
					case "google":
						if initNode.ddnsClient, err = google.New(c.DDNS.Config, n.Domain); err != nil {
							log.Panicln(err)
						}
					}
				}

				// init notifier
				if enableNoti {
					initNode.notifier = noti
				}

				s.nodes = append(s.nodes, initNode)
			}
		}
	}

	return s
}

func (s *Server) Start() {
	// On init start, do once check
	defer s.task()
	s.running = true

	// cron check
	if _, err := s.cron.AddFunc(fmt.Sprintf("@every %ds", s.internal), s.task); err != nil {
		log.Panic(err)
	}

	s.cron.Start()
	log.Warnln(config.AppName, "Started")
}

func (s *Server) task() {
	if s.cronRunning.Load() {
		return
	}

	s.cronRunning.Store(true)
	defer s.cronRunning.Store(false)

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

	s.handleBlockNodes()
}

func (s *Server) handleBlockNodes() {
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

			addr := fmt.Sprint(n.domain + ":" + strconv.Itoa(n.port))
			credValue, _ := n.svc.Config.Credentials.Get()

			// Get lightsail instance IP
			if n.ip == "" {
				inst, err := n.svc.GetInstance(&lightsail.GetInstanceInput{InstanceName: aws.String(n.name)})
				if err != nil {
					log.Error(err)
					return
				}
				switch n.network {
				case "tcp4":
					n.ip = aws.StringValue(inst.Instance.PublicIpAddress)
				case "tcp6":
					n.ip = aws.StringValue(inst.Instance.Ipv6Addresses[0])
				}
			}

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
		// Allocate Static Ip
		for svc := range svcMap {
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
