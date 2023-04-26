package config

type Config struct {
	LogLevel   string     `yaml:"LogLevel"`
	Internal   int        `yaml:"Internal"`
	Timeout    int        `yaml:"Timeout"`
	Nameserver string     `yaml:"Nameserver"`
	Concurrent int        `yaml:"Concurrent"`
	DDNS       *DDNS      `yaml:"DDNS"`
	Accounts   []*Account `yaml:"Accounts"`
}

type Account struct {
	AccessKeyID     string    `yaml:"AccessKeyID"`
	SecretAccessKey string    `yaml:"SecretAccessKey"`
	Regions         []*Region `yaml:"Regions"`
}

type Region struct {
	Name  string  `yaml:"Name"`
	Nodes []*Node `yaml:"Nodes"`
}

type Node struct {
	InstanceName string `yaml:"InstanceName"`
	Network      string `yaml:"Network"`
	Domain       string `yaml:"Domain"`
	Port         int    `yaml:"Port"`
}

type DDNS struct {
	Provider string
	DNSEnv   map[string]string
}
