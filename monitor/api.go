//    Copyright 2018 tub
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package monitor

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tkanos/gonfig"
)

type Configuration struct {
	configDir string //the directory where all config files are located

	Endpoint    string   // the endpoint that all requests are send to
	endpointURL *url.URL //internal URL represetantion

	ElasticSearchURL string //eleasticSerach endpoint

	VDCName string // VDCName (used for the index name in elastic serach)

	Opentracing    bool   //tells the proxy if a tracing header should be injected
	ZipkinEndpoint string //zipkin endpoint

	UseACME       bool //if true the proxy will aquire a LetsEncrypt certificate for the SSL connection
	UseSelfSigned bool //if UseACME is false, the proxy can use self signed certificates

	ForwardTraffic      bool //if true all traffic is forwareded to the exchangeReporter
	ExchangeReporterURL string
}

type meterMessage struct {
	RequestID string

	Timestamp time.Time `json:"@timestamp"`

	Client      string
	Method      string
	RequestTime time.Duration
}

type exchangeMessage struct {
	RequestID string

	Timestamp time.Time `json:"@timestamp"`

	RequestBody   string
	RequestHeader http.Header

	ResponseBody   string
	ResponseHeader http.Header
}

func readConfig(dir string) (Configuration, error) {
	configuration := Configuration{}
	err := gonfig.GetConf(fmt.Sprintf("%smonitor.json", dir), &configuration)
	if err != nil {
		log.Error("failed to load config", err)
		return configuration, err
	}

	url, err := url.Parse(configuration.Endpoint)
	if err != nil {
		log.Errorf("target URL could not be parsed", err)
		return configuration, err
	}

	configuration.endpointURL = url
	configuration.configDir = dir
	return configuration, nil
}