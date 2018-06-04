package monitor

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type exchangeReporter struct {
	Queue            chan exchangeMessage
	ExchangeEndpoint string
	QuitChan         chan bool
}

//newExchangeReporter creates a new exchange worker
func newExchangeReporter(ExchangeEndpoint string, queue chan exchangeMessage) (exchangeReporter, error) {
	//Wait for endpoint to become availible or timeout with error
	return exchangeReporter{
		Queue:            queue,
		ExchangeEndpoint: ExchangeEndpoint,
		QuitChan:         make(chan bool),
	}, nil
}

//Start will create a new worker process, for processing exchange Messages
func (er *exchangeReporter) Start() {
	go func() {
		for {

			select {
			case work := <-er.Queue:
				b := new(bytes.Buffer)
				json.NewEncoder(b).Encode(work)
				//send
				log.Debug("sending data to excahge!")
				resp, err := http.Post(er.ExchangeEndpoint, "application/json; charset=utf-8", b)

				if err != nil {
					log.Debug("failed to forward to :%s", err)
				}

				if resp.StatusCode > 200 {
					msg, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						log.Info("exchange failed %d - %s", resp.StatusCode, string(msg))
					}
				}
				log.Debugf("send data to excahge with: %d", resp.StatusCode)

			case <-er.QuitChan:
				// We have been asked to stop.
				log.Info("worker%d stopping")
				return
			}
		}
	}()
}

//Stop will terminate any running worker process
func (er *exchangeReporter) Stop() {
	go func() {
		er.QuitChan <- true
	}()
}
