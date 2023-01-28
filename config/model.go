package config

type Config struct {
	LogLevel string     `yaml:"LogLevel"`
	Internal int        `yaml:"Internal"`
	Timeout  int        `yaml:"Timeout"`
	Accounts []*Account `yaml:"Accounts"`
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
	Address string `yaml:"Address"`
	Port    int    `yaml:"Port"`
}
