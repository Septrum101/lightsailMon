package google

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

type Google struct {
	username string
	password string
	lastIpv4 string
	lastIpv6 string
	client   *resty.Client
}

func New(c map[string]string) (*Google, error) {
	g := &Google{
		username: c[strings.ToLower("GOOGLEDOMAIN_USERNAME")],
		password: c[strings.ToLower("GOOGLEDOMAIN_PASSWORD")],
	}

	cli := resty.New()
	cli.SetBasicAuth(g.username, g.password).SetBaseURL("https://domains.google.com")
	g.client = cli

	return g, nil
}

func (g *Google) AddUpdateDomainRecords(network string, ipAddr string, domain string) error {
	switch network {
	case "tcp4":
		return g.addUpdateDomainRecords("A", ipAddr, domain)
	case "tcp6":
		return g.addUpdateDomainRecords("AAAA", ipAddr, domain)
	default:
		return errors.New("not support network")
	}
}

func (g *Google) addUpdateDomainRecords(recordType string, ipAddr string, domain string) error {
	if ipAddr == "" {
		return errors.New("IP address is nil")
	}

	switch recordType {
	case "A":
		if g.lastIpv4 == ipAddr {
			return fmt.Errorf("[%s] IP %s have no change", domain, ipAddr)
		}

		if err := g.doRequest(ipAddr, domain); err != nil {
			return err
		}
		g.lastIpv4 = ipAddr
	case "AAAA":
		if g.lastIpv6 == ipAddr {
			return fmt.Errorf("[%s] IP %s have no change", domain, ipAddr)
		}

		if err := g.doRequest(ipAddr, domain); err != nil {
			return err
		}
		g.lastIpv6 = ipAddr
	}
	return nil
}

func (g *Google) doRequest(ipAddr string, domain string) error {
	resp, err := g.client.R().SetQueryParam("myip", ipAddr).
		SetQueryParams(map[string]string{
			"myip":     ipAddr,
			"hostname": domain,
		}).Get("/nic/update")
	if err != nil {
		return err
	}
	respStr := resp.String()
	if strings.Contains(respStr, "good") || strings.Contains(respStr, "nochg") {
		log.Printf("[%s] update record success, IP: %s", domain, ipAddr)
	} else {
		return fmt.Errorf("[%s] update record failure, Error: %s", domain, respStr)
	}
	return nil
}

func (g *Google) GetDomainRecords(recordType string, domain string) (domains map[string]bool, err error) {
	domains = make(map[string]bool)
	switch recordType {
	case "A":
		domains[g.lastIpv4] = true
	case "AAAA":
		domains[g.lastIpv6] = true
	}

	return nil, errors.New("no record")
}
