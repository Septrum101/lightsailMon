package ddns

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

type GoogleDomain struct {
	domain   string
	username string
	password string
	lastIpv4 string
	lastIpv6 string
	client   *resty.Client
}

func (g *GoogleDomain) Init(c map[string]string, d string) error {
	g.domain = d
	g.username = c[strings.ToLower("GOOGLEDOMAIN_USERNAME")]
	g.password = c[strings.ToLower("GOOGLEDOMAIN_PASSWORD")]

	cli := resty.New()
	cli.SetBasicAuth(g.username, g.password).SetBaseURL("https://domains.google.com").
		SetQueryParam("hostname", g.domain)
	g.client = cli

	return nil
}

func (g *GoogleDomain) AddUpdateDomainRecords(network string, ipAddr string) error {
	switch network {
	case "tcp4":
		return g.addUpdateDomainRecords("A", ipAddr)
	case "tcp6":
		return g.addUpdateDomainRecords("AAAA", ipAddr)
	default:
		return errors.New("not support network")
	}
}

func (g *GoogleDomain) addUpdateDomainRecords(recordType string, ipAddr string) error {
	if ipAddr == "" {
		return errors.New("IP address is nil")
	}

	switch recordType {
	case "A":
		if g.lastIpv4 == ipAddr {
			return fmt.Errorf("[%s] IP %s have no change", g.domain, ipAddr)
		}

		if err := g.doRequest(ipAddr); err != nil {
			return err
		}
		g.lastIpv4 = ipAddr
	case "AAAA":
		if g.lastIpv6 == ipAddr {
			return fmt.Errorf("[%s] IP %s have no change", g.domain, ipAddr)
		}

		if err := g.doRequest(ipAddr); err != nil {
			return err
		}
		g.lastIpv6 = ipAddr
	}
	return nil
}

func (g *GoogleDomain) doRequest(ipAddr string) error {
	resp, err := g.client.R().SetQueryParam("myip", ipAddr).Get("/nic/update")
	if err != nil {
		return err
	}
	respStr := resp.String()
	if strings.Contains(respStr, "good") || strings.Contains(respStr, "nochg") {
		log.Printf("[%s] update record success, IP: %s", g.domain, ipAddr)
	} else {
		return fmt.Errorf("[%s] update record failure, Error: %s", g.domain, respStr)
	}
	return nil
}
