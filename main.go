package main

import (
	"flag"

	"github.com/DITAS-Project/VDC-Request-Monitor/monitor"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var (
	Build string
)

var logger = logrus.New()
var log *logrus.Entry

func init() {
	if Build == "" {
		Build = "Debug"
	}
	logger.Formatter = new(prefixed.TextFormatter)
	logger.SetLevel(logrus.DebugLevel)
	log = logger.WithFields(logrus.Fields{
		"prefix": "req-mon",
		"build":  Build,
	})
}

func main() {
	viper.SetConfigName("monitor")
	viper.AddConfigPath("/opt/blueprint/")
	viper.AddConfigPath("/.config/")
	viper.AddConfigPath(".config/")
	viper.AddConfigPath(".")

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
	flag.Bool("verbose", false, "for verbose logging")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	if viper.GetBool("verbose") {
		logger.SetLevel(logrus.DebugLevel)
	}

	monitor.SetLogger(logger)
	monitor.SetLog(log)

	//
	mon, err := monitor.NewManger()

	if err != nil {
		log.Fatalf("could not start request monitor %+v", err)
	}

	mon.Listen()
}
