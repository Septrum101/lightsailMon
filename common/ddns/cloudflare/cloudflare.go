package cloudflare

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// Cloudflare Implementation
type Cloudflare struct {
	domain string
	client *cloudflare.API
}

func New(c map[string]string, d string) (*Cloudflare, error) {
	cf := &Cloudflare{
		domain: d,
	}

	client, err := cloudflare.New(c[strings.ToLower("CLOUDFLARE_API_KEY")], c[strings.ToLower("CLOUDFLARE_EMAIL")])
	if err != nil {
		return nil, err
	}
	cf.client = client
	return cf, nil
}

// AddUpdateDomainRecords create or update IPv4/IPv6 records
func (cf *Cloudflare) AddUpdateDomainRecords(network string, ipAddr string) error {
	switch network {
	case "tcp4":
		return cf.addUpdateDomainRecords("A", ipAddr)
	case "tcp6":
		return cf.addUpdateDomainRecords("AAAA", ipAddr)
	default:
		return errors.New("not support network")
	}
}

func (cf *Cloudflare) addUpdateDomainRecords(recordType string, ipAddr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if ipAddr == "" {
		return errors.New("IP address is nil")
	}

	zones, err := cf.client.ListZones(ctx)
	if err != nil {
		return err
	}

	// Get zone
	zoneID := ""
	for i := range zones {
		if strings.Contains(cf.domain, zones[i].Name) {
			zoneID = zones[i].ID
		}
	}
	if zoneID == "" {
		return errors.New("cannot find a valid zone")
	}

	records, _, err := cf.client.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{
		Type: recordType,
		Name: cf.domain,
	})
	if err != nil {
		return err
	}
	if len(records) > 0 {
		for i := range records {
			if records[i].Content == ipAddr {
				return fmt.Errorf("[%s] IP %s have no change", cf.domain, ipAddr)
			}
			_, err = cf.client.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.UpdateDNSRecordParams{
				Type:    recordType,
				ID:      records[i].ID,
				Content: ipAddr,
			})
			if err != nil {
				return fmt.Errorf("[%s] update record failure, Error: %s", cf.domain, err)
			}
			log.Printf("[%s] update record success, IP: %s", cf.domain, ipAddr)
		}
	} else {
		_, err := cf.client.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.CreateDNSRecordParams{
			Type:    recordType,
			Name:    cf.domain,
			Content: ipAddr,
		})
		if err != nil {
			return fmt.Errorf("[%s] create record failure, Error: %s", cf.domain, err)
		}
		log.Printf("[%s] create record success, IP: %s", cf.domain, ipAddr)
	}
	return nil
}
