package controller

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	"github.com/thank243/lightsailMon/app"
	"github.com/thank243/lightsailMon/app/dns"
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
		fmt.Printf("Log level: %s  (Concurrent: %d)\n", c.LogLevel, c.Concurrent)
	}

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

			// create lightsail svc
			svc := svc{
				RWMutex:   new(sync.RWMutex),
				Lightsail: lightsail.New(sess, aws.NewConfig().WithRegion(r.Name)),
			}

			for iii := range r.Nodes {
				n := r.Nodes[iii]
				s.nodes = append(s.nodes, &node{
					network:    n.Network,
					address:    n.Address,
					port:       n.Port,
					svc:        svc,
					nameserver: dns.New(n.Network, c.Nameserver),
				})
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
	resp, err := http.Get("http://www.gstatic.com/generate_204")
	if err != nil {
		log.Error(err)
		return
	}
	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		log.Error(string(body))
		return
	}

	s.handleBlockNodes()
}

func (s *Server) handleBlockNodes() {
	var blockNodes []*node
	svcMap := make(map[svc]uint8)

	for k := range s.nodes {
		node := s.nodes[k]

		// The next change IP must more than 10min
		if !time.Now().After(node.lastChangeIP.Add(time.Minute * 10)) {
			log.Infof("[%s:%d] The last IP change period time less than 10min", node.address, node.port)
			continue
		}

		s.wg.Add(1)
		s.worker <- 0

		go func() {
			defer func() {
				<-s.worker
				s.wg.Done()
			}()

			addr := fmt.Sprint(node.address + ":" + strconv.Itoa(node.port))
			credValue, _ := node.svc.Config.Credentials.Get()

			// resolve ips for domain
			ips := node.nameserver.LookupIP(node.address)
			flag := false
			for i := range ips {
				start := time.Now()
				if err := app.CheckConnection(ips[i], node.port, s.timeout, node.network); err != nil {
					if v, ok := err.(*net.OpError); ok && v.Addr != nil {
						flag = true
					}
					log.Errorf("[AccessKeyID: %s] %s %v", credValue.AccessKeyID, addr, err)
				} else {
					log.Infof("[AccessKeyID: %s] %s Tcping: %d ms", credValue.AccessKeyID, addr, time.Since(start).Milliseconds())
					flag = false
					break
				}
			}

			if flag {
				s.Lock()
				defer s.Unlock()
				// add to blockNodes
				blockNodes = append(blockNodes, node)
				svcMap[node.svc] = 0
			}
		}()
	}
	s.wg.Wait()

	if len(blockNodes) > 0 {
		instanceMap := make(map[string]string)
		for svc := range svcMap {
			pageToken := ""
			for {
				ins, err := svc.GetInstances(&lightsail.GetInstancesInput{
					PageToken: aws.String(pageToken),
				})
				if err != nil {
					log.Error(err)
				}

				// create map: {ip: name}
				for i := range ins.Instances {
					inst := ins.Instances[i]
					if inst != nil {
						instanceMap[*inst.PublicIpAddress] = *inst.Name
					}
				}

				// update pageToken or break loop
				if ins.NextPageToken != nil {
					pageToken = *ins.NextPageToken
				} else {
					break
				}
			}

			// Allocate Static Ip
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
			n := blockNodes[i]

			go func() {
				defer func() {
					<-s.worker
					s.wg.Done()
				}()

				log.Errorf("[%s:%d] Change node IP", n.address, n.port)

				n.changeIP(instanceMap)
				n.lastChangeIP = time.Now()
			}()
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
	s.running = false
}
