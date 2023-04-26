package ddns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/thank243/lightsailMon/config"
)

const (
	zonesAPI string = "https://api.cloudflare.com/client/v4/zones"
)

// Cloudflare implement
type Cloudflare struct {
	Domain string
	Secret string
}

// CloudflareZonesResp cloudflare zones result
type CloudflareZonesResp struct {
	CloudflareStatus
	Result []struct {
		ID     string
		Name   string
		Status string
		Paused bool
	}
}

// CloudflareRecordsResp records response
type CloudflareRecordsResp struct {
	CloudflareStatus
	Result []CloudflareRecord
}

// CloudflareRecord records
type CloudflareRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

// CloudflareStatus public status
type CloudflareStatus struct {
	Success  bool
	Messages []string
}

func (cf *Cloudflare) Init(c *config.DNS, d string) {
	cf.Domain = d
	cf.Secret = c.Secret
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

	if ipAddr == "" {
		return
	}

	// get zone
	zoneID := ""
	zoneName := cf.Domain
	for {
		dList := strings.Split(zoneName, ".")
		if len(dList) == 1 {
			break
		}
		result, err := cf.getZones(zoneName)
		if err != nil || len(result.Result) != 1 {
			zoneName = strings.Join(dList[1:], ".")
			continue
		}
		zoneID = result.Result[0].ID
		break
	}
	if zoneID == "" {
		log.Error("cannot find a valid zone")
		return
	}

	var records CloudflareRecordsResp
	// getDomains max records is 50
	err := cf.request(
		"GET",
		fmt.Sprintf(zonesAPI+"/%s/dns_records?type=%s&name=%s&per_page=50", zoneID, recordType, cf.Domain),
		nil,
		&records,
	)

	if err != nil || !records.Success {
		log.Error(err)
		return
	}

	if len(records.Result) > 0 {
		// update
		cf.modify(records, zoneID, cf.Domain, ipAddr)
	} else {
		// create new record
		cf.create(zoneID, cf.Domain, recordType, ipAddr)
	}

}

// create records
func (cf *Cloudflare) create(zoneID string, domain string, recordType string, ipAddr string) {
	record := &CloudflareRecord{
		Type:    recordType,
		Name:    domain,
		Content: ipAddr,
	}
	var status CloudflareStatus
	err := cf.request(
		"POST",
		fmt.Sprintf(zonesAPI+"/%s/dns_records", zoneID),
		record,
		&status,
	)
	if err == nil && status.Success {
		log.Printf("Create record %s success! IP: %s", domain, ipAddr)
	} else {
		log.Printf("Create record %s failure! Messages: %s", domain, status.Messages)
	}
}

// update records
func (cf *Cloudflare) modify(result CloudflareRecordsResp, zoneID string, domain string, ipAddr string) {
	for _, record := range result.Result {
		if record.Content == ipAddr {
			log.Printf("Your IP %s have no change, domain %s", ipAddr, domain)
			continue
		}

		var status CloudflareStatus
		record.Content = ipAddr
		err := cf.request(
			"PUT",
			fmt.Sprintf(zonesAPI+"/%s/dns_records/%s", zoneID, record.ID),
			record,
			&status,
		)
		if err == nil && status.Success {
			log.Printf("Create record %s success! IP: %s", domain, ipAddr)
		} else {
			log.Printf("Create record %s failure! Messages: %s", domain, status.Messages)
		}
	}
}

// get records list
func (cf *Cloudflare) getZones(domain string) (result CloudflareZonesResp, err error) {
	err = cf.request(
		"GET",
		fmt.Sprintf(zonesAPI+"?name=%s&status=%s&per_page=%s", domain, "active", "50"),
		nil,
		&result,
	)

	return
}

func (cf *Cloudflare) request(method string, url string, data interface{}, result interface{}) (err error) {
	jsonStr := make([]byte, 0)
	if data != nil {
		jsonStr, _ = json.Marshal(data)
	}
	req, err := http.NewRequest(
		method,
		url,
		bytes.NewBuffer(jsonStr),
	)
	if err != nil {
		log.Println("http.NewRequest failure. Error: ", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+cf.Secret)
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	json.Unmarshal(body, &result)

	if resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("request %s failure! content: %s , status code: %d\n", url, string(body), resp.StatusCode)
		log.Println(errMsg)
		err = fmt.Errorf(errMsg)
	}

	return
}
