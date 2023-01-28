package config

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

var (
	viperOnce sync.Once
	v         *viper.Viper
)

func GetConfig() *viper.Viper {
	viperOnce.Do(func() {
		v = viper.New()
		v.SetConfigName("config")
		v.SetConfigType("yml")
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/" + AppName)
		v.AddConfigPath("$HOME/." + AppName)

		if err := v.ReadInConfig(); err != nil {
			log.Panic(err)
		}
	})

	return v
}
