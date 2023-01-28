package app

import (
	"io"
	"net/http"
	"time"

	"github.com/bitly/go-simplejson"
	log "github.com/sirupsen/logrus"
)

func LookupIP(name string) []string {
	client := http.Client{
		Timeout: time.Second * 20,
	}

	req, err := http.NewRequest("GET", "https://doh.pub/dns-query", nil)
	if err != nil {
		log.Error(err)
	}

	req.Header.Add("accept", "application/dns-json")

	q := req.URL.Query()
	q.Add("name", name)
	q.Add("type", "1")
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		log.Error(err)
		return nil
	}

	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	json, err := simplejson.NewJson(body)
	if err != nil {
		return nil
	}
	ipList := json.Get("Answer").MustArray()
	if len(ipList) == 0 {
		return nil
	}

	var ips []string
	for i := range ipList {
		ip := ipList[i].(map[string]any)
		if i, ok := ip["data"]; ok {
			ips = append(ips, i.(string))
		}
	}
	return ips
}
