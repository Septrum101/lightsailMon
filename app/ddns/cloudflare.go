package ddns

import (
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/thank243/lightsailMon/config"
)

// Cloudflare Implementation
type Cloudflare struct {
	domain string
	client *cloudflare.API
}

func (cf *Cloudflare) Init(c *config.DDNS, d string) error {
	cf.domain = d
	client, err := cloudflare.New(c.DNSEnv[strings.ToLower("CLOUDFLARE_API_KEY")], c.DNSEnv[strings.ToLower("CLOUDFLARE_EMAIL")])
	if err != nil {
		return err
	}
	cf.client = client
	return nil
}

// AddUpdateDomainRecords create or update IPv4/IPv6 records
func (cf *Cloudflare) AddUpdateDomainRecords(network string, ipAddr string) {
	switch network {
	case "tcp4":
		cf.addUpdateDomainRecords("A", ipAddr)
	case "tcp6":
		cf.addUpdateDomainRecords("AAAA", ipAddr)
	}
}

func (cf *Cloudflare) addUpdateDomainRecords(recordType string, ipAddr string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if ipAddr == "" {
		return
	}

	zones, err := cf.client.ListZones(ctx)
	if err != nil {
		log.Error(err)
		return
	}

	// Get zone
	zoneID := ""
	for i := range zones {
		if strings.Contains(cf.domain, zones[i].Name) {
			zoneID = zones[i].ID
		}
	}
	if zoneID == "" {
		log.Error("cannot find a valid zone")
		return
	}

	records, _, err := cf.client.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{
		Type: recordType,
		Name: cf.domain,
	})
	if err != nil {
		log.Error(err)
		return
	}
	if len(records) > 0 {
		for i := range records {
			if records[i].Content == ipAddr {
				log.Warnf("Your IP %s have no change, domain %s", ipAddr, cf.domain)
				return
			}
			_, err = cf.client.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.UpdateDNSRecordParams{
				Type:    recordType,
				ID:      records[i].ID,
				Content: ipAddr,
			})
			if err != nil {
				log.Errorf("Update record %s failure, Error: %s", cf.domain, err)
				return
			}
			log.Printf("Update record %s success, IP: %s", cf.domain, ipAddr)
		}
	} else {
		_, err := cf.client.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.CreateDNSRecordParams{
			Type:    recordType,
			Name:    cf.domain,
			Content: ipAddr,
		})
		if err != nil {
			log.Errorf("Create record %s failure, Error: %s", cf.domain, err)
			return
		}
		log.Printf("Create record %s success, IP: %s", cf.domain, ipAddr)
	}
}
