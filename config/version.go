package config

import (
	"fmt"
)

var (
	version = "dev"
	AppName = "LightsailMon"
	intro   = "An AWS Lightsail monitor service that can auto change blocked IP."
	date    = "unknown"
)

func ShowVersion() {
	fmt.Printf("%s %s, built at %s\n%s\n", AppName, version, date, intro)
}
