package ddns

type Client interface {
	Init(c map[string]string, d string) error
	AddUpdateDomainRecords(network string, ipAddr string) error
}
