package ddns

type Client interface {
	AddUpdateDomainRecords(network string, ipAddr string) error
}
