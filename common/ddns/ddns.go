package ddns

type Client interface {
	AddUpdateDomainRecords(network string, ipAddr string) error
	GetDomainRecords(recordType string) (domains map[string]bool, err error)
}
