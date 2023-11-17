package controller

import (
	log "github.com/sirupsen/logrus"

	"github.com/thank243/lightsailMon/app/node"
	"github.com/thank243/lightsailMon/common/ddns"
	"github.com/thank243/lightsailMon/common/ddns/cloudflare"
	"github.com/thank243/lightsailMon/common/ddns/google"
	"github.com/thank243/lightsailMon/common/notify"
)

func (s *Service) buildNodes(isNotify bool, isDDNS bool) []*node.Node {
	// init notifier
	var notifier notify.Notify
	if isNotify {
		switch s.conf.Notify.Provider {
		case "pushplus":
			notifier = &notify.PushPlus{Token: s.conf.Notify.Config["pushplus_token"].(string)}
		case "telegram":
			notifier = &notify.Telegram{
				ChatID: int64(s.conf.Notify.Config["telegram_chatid"].(int)),
				Token:  s.conf.Notify.Config["telegram_token"].(string),
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
		newNode := node.New(s.conf.Nodes[i])

		// set ddns client
		if isDDNS {
			newNode.SetDDNSClient(ddnsCli)
		}

		// set notifier
		if isNotify {
			newNode.SetNotifier(notifier)
		}

		// set connection timeout
		if s.conf.Timeout > 0 {
			newNode.SetTimeout(s.conf.Timeout)
		}

		nodes = append(nodes, newNode)
	}

	return nodes
}
