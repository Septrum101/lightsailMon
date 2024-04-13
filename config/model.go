package config

type Config struct {
	LogLevel   string
	Internal   int
	Timeout    int
	Nameserver string
	Concurrent int
	Ipv6       bool
	DDNS       *DDNS
	Notify     *Notify
	Nodes      []*Node
}

type Node struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	InstanceName    string
	Network         []string
	Domain          string
	Port            int
}

type DDNS struct {
	Enable   bool
	Provider string
	Config   map[string]string
}

type Notify struct {
	Enable   bool
	Provider string
	Config   map[string]any
}
