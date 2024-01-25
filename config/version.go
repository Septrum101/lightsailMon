package config

import (
	"fmt"
)

var (
	Version = "0.2.8"
	AppName = "LightsailMon"
	Intro   = "An AWS Lightsail monitor service that can auto change blocked IP."
	date    = "unknown"
)

func ShowVersion() {
	fmt.Printf("%s %s, built at %s\n%s\n", AppName, Version, date, Intro)
}
