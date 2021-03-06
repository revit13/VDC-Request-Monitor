/*
 * Copyright 2018 Information Systems Engineering, TU Berlin, Germany
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *                       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * This is being developed for the DITAS Project: https://www.ditas-project.eu/
 */

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
