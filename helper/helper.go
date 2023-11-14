package helper

import (
	"net"

	log "github.com/sirupsen/logrus"
)

func GetDomainIP(network string, domain string) string {
	ips, err := net.LookupIP(domain)
	if err != nil {
		log.Error(err)
		return ""
	}
	if len(ips) == 0 {
		return ""
	}

	ip := ""
	for i := range ips {
		switch network {
		case "tcp4":
			ipv4 := ips[i].To4()
			if ipv4 != nil {
				ip = ipv4.String()
				break
			}

		case "tcp6":
			ipv6 := ips[i].To16()
			if ipv6 != nil {
				ip = ipv6.String()
				break
			}
		}
	}

	return ip
}
