package main

import (
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	servers "github.com/OverlayFox/VRC-Stream-Haven/servers"
	"github.com/oschwald/geoip2-golang"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	nodes []struct {
		publicIpAddress string
		publicPort      string
		latitude        float64
		longitude       float64
	}
}

var ipDb geoip2.Reader
var config Config
var log = logger.Logger

func init() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Could not read config file")
		return
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Error unmarshaling config")
		return
	}
}

func main() {
	s := &servers.RtspServer{}
	s.Initialize()

}
