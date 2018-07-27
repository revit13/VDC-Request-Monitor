package main

import (
	"flag"

	"github.com/DITAS-Project/VDC-Request-Monitor/monitor"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("monitor")
	viper.AddConfigPath("/.config/")
	viper.AddConfigPath("/opt/blueprint/")
	viper.AddConfigPath(".")
	viper.AddConfigPath(".config/")
	//setup defaults
	viper.SetDefault("Endpoint", "http://localhost:8080")
	viper.SetDefault("ElasticSearchURL", "http://localhost:9200")
	viper.SetDefault("VDCName", "dummyVDC")
	viper.SetDefault("Opentracing", false)
	viper.SetDefault("ZipkinEndpoint", "")
	viper.SetDefault("UseACME", false)
	viper.SetDefault("UseSelfSigned", true)
	viper.SetDefault("ForwardTraffic", false)
	viper.SetDefault("ExchangeReporterURL", "")

	//setup cmd interface
	flag.String("elastic", viper.GetString("ElasticSearchURL"), "used to define the elasticURL")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	log.SetLevel(log.DebugLevel)

	//
	mon, err := monitor.NewManger()

	if err != nil {
		log.Fatalf("could not start request monitor %+v", err)
	}

	mon.Listen()
}
