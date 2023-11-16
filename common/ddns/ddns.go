package ddns

type Client interface {
	AddUpdateDomainRecords(network string, ipAddr string, domain string) error
	GetDomainRecords(recordType string, domain string) (domains map[string]bool, err error)
}
