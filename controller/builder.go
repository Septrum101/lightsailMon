package controller

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Septrum101/lightsailMon/app/node"
	"github.com/Septrum101/lightsailMon/common/ddns"
	"github.com/Septrum101/lightsailMon/common/ddns/cloudflare"
	"github.com/Septrum101/lightsailMon/common/ddns/google"
	"github.com/Septrum101/lightsailMon/common/notify"
	"github.com/Septrum101/lightsailMon/common/notify/pushplus"
	"github.com/Septrum101/lightsailMon/common/notify/telegram"
)

func (s *Service) buildNodes(isNotify bool, isDDNS bool) []*node.Node {
	// init notifier
	var notifier notify.Notify
	if isNotify {
		switch s.conf.Notify.Provider {
		case "pushplus":
			notifier = &pushplus.PushPlus{Token: s.conf.Notify.Config["pushplus_token"]}
		case "telegram":
			notifier = &telegram.Telegram{
				ApiHost: s.conf.Notify.Config["telegram_apihost"],
				ChatID:  s.conf.Notify.Config["telegram_chatid"],
				Token:   s.conf.Notify.Config["telegram_token"],
			}
		}
	}

	// init ddnsCli
	var ddnsCli ddns.Client
	if isDDNS {
		var err error
		switch s.conf.DDNS.Provider {
		case "cloudflare":
			if ddnsCli, err = cloudflare.New(s.conf.DDNS.Config); err != nil {
				log.Panicln(err)
			}
		case "google":
			if ddnsCli, err = google.New(s.conf.DDNS.Config); err != nil {
				log.Panicln(err)
			}
		}
	}

	var nodes []*node.Node
	for i := range s.conf.Nodes {
		newNodes := node.New(s.conf.Nodes[i])
		for ii := range newNodes {
			newNode := newNodes[ii]
			// set ddns client
			if isDDNS {
				newNode.DdnsClient = ddnsCli
			}

			// set notifier
			if isNotify {
				newNode.Notifier = notifier
			}

			// set connection timeout
			if s.conf.Timeout > 0 {
				newNode.Timeout = time.Second * time.Duration(s.conf.Timeout)
			}

			nodes = append(nodes, newNode)
		}
	}

	return nodes
}
