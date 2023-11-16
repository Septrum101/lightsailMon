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
	client *cloudflare.API
}

func New(c map[string]string) (*Cloudflare, error) {
	cf := new(Cloudflare)

	client, err := cloudflare.New(c[strings.ToLower("CLOUDFLARE_API_KEY")], c[strings.ToLower("CLOUDFLARE_EMAIL")])
	if err != nil {
		return nil, err
	}
	cf.client = client

	return cf, nil
}

// AddUpdateDomainRecords create or update IPv4/IPv6 records
func (cf *Cloudflare) AddUpdateDomainRecords(network string, domain string, ipAddr string) error {
	switch network {
	case "tcp4":
		return cf.addUpdateDomainRecords("A", domain, ipAddr)
	case "tcp6":
		return cf.addUpdateDomainRecords("AAAA", domain, ipAddr)
	default:
		return errors.New("not support network")
	}
}

func (cf *Cloudflare) addUpdateDomainRecords(recordType string, domain string, ipAddr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if ipAddr == "" {
		return errors.New("IP address is nil")
	}

	zoneID, records, err := cf.getRecords(ctx, recordType, domain)
	if err != nil {
		return err
	}

	if len(records) > 0 {
		for i := range records {
			if records[i].Content == ipAddr {
				return fmt.Errorf("[%s] IP %s have no change", domain, ipAddr)
			}
			_, err = cf.client.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.UpdateDNSRecordParams{
				Type:    recordType,
				ID:      records[i].ID,
				Content: ipAddr,
			})
			if err != nil {
				return fmt.Errorf("[%s] update record failure, Error: %s", domain, err)
			}
			log.Printf("[%s] update record success, IP: %s", domain, ipAddr)
		}
	} else {
		_, err := cf.client.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.CreateDNSRecordParams{
			Type:    recordType,
			Name:    domain,
			Content: ipAddr,
		})
		if err != nil {
			return fmt.Errorf("[%s] create record failure, Error: %s", domain, err)
		}
		log.Printf("[%s] create record success, IP: %s", domain, ipAddr)
	}
	return nil
}

func (cf *Cloudflare) getRecords(ctx context.Context, recordType string, domain string) (string, []cloudflare.DNSRecord, error) {
	zones, err := cf.client.ListZones(ctx)
	if err != nil {
		return "", nil, err
	}

	// Get zone
	zoneID := ""
	for i := range zones {
		if strings.Contains(domain, zones[i].Name) {
			zoneID = zones[i].ID
		}
	}
	if zoneID == "" {
		return "", nil, errors.New("cannot find a valid zone")
	}

	records, _, err := cf.client.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{
		Type: recordType,
		Name: domain,
	})
	if err != nil {
		return "", nil, err
	}
	return zoneID, records, nil
}

func (cf *Cloudflare) GetDomainRecords(recordType string, domain string) (domains map[string]bool, err error) {
	domains = make(map[string]bool)
	ctx := context.Background()
	_, records, err := cf.getRecords(ctx, recordType, domain)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for i := range records {
		domains[records[i].Content] = true
	}
	return domains, nil
}
