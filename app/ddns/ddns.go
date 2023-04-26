package ddns

import (
	"github.com/thank243/lightsailMon/config"
)

type Client interface {
	Init(c *config.DDNS, d string) error
	AddUpdateDomainRecords(network string, ipAddr string)
}
