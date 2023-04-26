package ddns

import (
	"github.com/thank243/lightsailMon/config"
)

type DDNS interface {
	Init(c *config.DNS, d string)
	AddUpdateDomainRecords(network string, ipAddr string)
}
