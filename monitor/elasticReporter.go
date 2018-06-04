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
	"context"
	"fmt"
	"time"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"
)

type elasticReporter struct {
	Queue    chan meterMessage
	Client   *elastic.Client
	VDCName  string
	QuitChan chan bool
	ctx      context.Context
}

//newElasticReporter creates a new reporter worker,
//will fail if no elastic client can be built
//otherwise retunrs a worker handler
func newElasticReporter(config Configuration, queue chan meterMessage) (elasticReporter, error) {
	//Wait for endpoint to become availible or timeout with error

	client, err := elastic.NewClient(
		elastic.SetURL(config.ElasticSearchURL),
	)

	if err != nil {
		log.Error("failed to connect to elastic serach")
		return elasticReporter{}, err
	}

	reporter := elasticReporter{
		Queue:    queue,
		Client:   client,
		VDCName:  config.VDCName,
		QuitChan: make(chan bool),
		ctx:      context.Background(),
	}

	return reporter, nil
}

//Start creates a new worker process and waits for meterMessages
//can only be terminated by calling Stop()
func (er *elasticReporter) Start() {
	go func() {
		for {

			select {
			case work := <-er.Queue:
				//TODO
				log.Infof("reporting %s - %s", work.Client, work.Method)

				work.Timestamp = time.Now()

				_, err := er.Client.Index().Index(er.getElasticIndex()).Type("request-monitor").BodyJson(work).Do(er.ctx)

				if err != nil {
					log.Debug("failed to report mesurement to :%s", err)
				} else {
					log.Debug("reported data to elastic!")
				}

			case <-er.QuitChan:
				// We have been asked to stop.
				log.Info("worker%d stopping")
				return
			}
		}
	}()
}

//Stop termintates this Worker
func (er *elasticReporter) Stop() {
	go func() {
		er.QuitChan <- true
	}()
}

func (er *elasticReporter) getElasticIndex() string {
	t := time.Now()
	return fmt.Sprintf("%s-%d-%02d-%02d", er.VDCName, t.Year(), t.Month(), t.Day())
}