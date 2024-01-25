package config

import (
	"fmt"
)

var (
	Version = "0.2.7"
	AppName = "LightsailMon"
	Intro   = "An AWS Lightsail monitor service that can auto change blocked IP."
)

func ShowVersion() {
	fmt.Printf("%s %s\n%s\n", AppName, Version, Intro)
}
