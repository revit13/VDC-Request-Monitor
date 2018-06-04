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
	Endpoint    string
	EndpointURL *url.URL

	ExchangeReporterURL string
	ElasticSearchURL    string

	VDCName string

	Opentracing    bool
	ZipkinEndpoint string

	UseACME       bool
	UseSelfSigned bool

	configDir string
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

	configuration.EndpointURL = url
	configuration.configDir = dir
	return configuration, nil
}
