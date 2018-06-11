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
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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

	Client      string        `json:"request.client"`
	Method      string        `json:"request.method"`
	RequestTime time.Duration `json:"request.requestTime"`
}

type exchangeMessage struct {
	RequestID string

	Timestamp time.Time `json:"@timestamp"`

	RequestBody   string
	RequestHeader http.Header

	ResponseBody   string
	ResponseHeader http.Header
}

func readConfig() (Configuration, error) {

	err := viper.ReadInConfig()
	configuration := Configuration{}
	if err != nil {
		log.Error("failed to load config", err)
		return configuration, err
	}

	viper.Unmarshal(&configuration)

	url, err := url.Parse(configuration.Endpoint)
	if err != nil {
		log.Errorf("target URL could not be parsed", err)
		return configuration, err
	}

	configuration.endpointURL = url
	configuration.configDir = filepath.Dir(viper.ConfigFileUsed())
	log.Infof("using this config %+v", configuration)
	return configuration, nil
}
