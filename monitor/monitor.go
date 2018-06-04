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
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"golang.org/x/crypto/acme/autocert"

	"github.com/kabukky/httpscerts"

	"github.com/vulcand/oxy/forward"
	"github.com/vulcand/oxy/utils"

	"github.com/satori/go.uuid"

	log "github.com/sirupsen/logrus"
)

//RequestMonitor data struct
type RequestMonitor struct {
	conf Configuration
	oxy  *forward.Forwarder

	monitorQueue  chan meterMessage
	exchangeQueue chan exchangeMessage

	reporter elasticReporter
	exporter exchangeReporter
}

//NewManger Creates a new logging, tracing RequestMonitor
func NewManger(confdir string) (*RequestMonitor, error) {

	configuration, err := readConfig(confdir)
	if err != nil {
		log.Error("could not read config!")
		return nil, err
	}

	mng := &RequestMonitor{
		conf:          configuration,
		monitorQueue:  make(chan meterMessage, 10),
		exchangeQueue: make(chan exchangeMessage, 10),
	}

	err = mng.initTracing()
	if err != nil {
		log.Errorf("failed to init tracer %+v", err)
	}

	fwd, err := forward.New(
		forward.Stream(true),
		forward.PassHostHeader(true),
		forward.ErrorHandler(utils.ErrorHandlerFunc(handleError)),
		forward.StateListener(forward.UrlForwardingStateListener(stateListener)),
		forward.ResponseModifier(mng.responseInterceptor),
	)

	if err != nil {
		log.Errorf("failed to init oxy %+v", err)
		return nil, err
	}

	mng.oxy = fwd

	reporter, err := newElasticReporter(configuration, mng.monitorQueue)
	if err != nil {
		log.Errorf("Failed to init elastic reporter %+v", err)
		return nil, err
	}
	mng.reporter = reporter

	if configuration.ForwardTraffic {
		exporter, err := newExchangeReporter(configuration.ExchangeReporterURL, mng.exchangeQueue)
		if err != nil {
			log.Errorf("Failed to init exchange reporter %+v", err)
			return nil, err
		}
		mng.exporter = exporter
	}

	log.Info("Request-Monitor created")

	return mng, nil
}

func stateListener(url *url.URL, state int) {
	if url != nil {
		log.Printf("url:%s - state:%d", url.String(), state)
	}
}

func handleError(w http.ResponseWriter, req *http.Request, err error) {
	statusCode := http.StatusInternalServerError
	if e, ok := err.(net.Error); ok {
		if e.Timeout() {
			statusCode = http.StatusGatewayTimeout
		} else {
			statusCode = http.StatusBadGateway
		}
	} else if err == io.EOF {
		statusCode = http.StatusBadGateway
	}

	log.Errorf("reqest:%s suffered internal error:%d - %v+", req.URL, statusCode, err)

	w.WriteHeader(statusCode)
	w.Write([]byte(http.StatusText(statusCode)))
}

func (mon *RequestMonitor) generateRequestID(remoteAddr string) string {
	return uuid.NewV5(uuid.NamespaceX500, remoteAddr).String()
}

func (mon *RequestMonitor) initTracing() error {
	if mon.conf.Opentracing {
		log.Info("opentracing active")
		// Create our HTTP collector.
		collector, err := zipkin.NewHTTPCollector(mon.conf.ZipkinEndpoint)
		if err != nil {
			log.Errorf("unable to create Zipkin HTTP collector: %+v\n", err)
			return err
		}

		// Create our recorder.
		recorder := zipkin.NewRecorder(collector, false, "0.0.0.0:0", "request-monitor")

		// Create our tracer.
		tracer, err := zipkin.NewTracer(
			recorder,
			zipkin.ClientServerSameSpan(true),
			zipkin.TraceID128Bit(true),
		)
		if err != nil {
			log.Errorf("unable to create Zipkin tracer: %+v\n", err)
			return err
		}

		// Explicitly set our tracer to be the default tracer.
		opentracing.InitGlobalTracer(tracer)
	}
	return nil
}

func (mon *RequestMonitor) push(requestID string, message meterMessage) {
	message.RequestID = requestID
	message.Timestamp = time.Now()
	mon.monitorQueue <- message
}

func (mon *RequestMonitor) forward(requestID string, message exchangeMessage) {
	if mon.conf.ForwardTraffic {
		message.RequestID = requestID
		message.Timestamp = time.Now()
		mon.exchangeQueue <- message
	}
}

//Listen will start all worker threads and wait for incoming requests
func (mon *RequestMonitor) Listen() {

	//start parallel reporter threads
	mon.reporter.Start()

	if mon.conf.ForwardTraffic {
		mon.exporter.Start()
		defer mon.exporter.Stop()
	}

	defer mon.reporter.Stop()

	var m *autocert.Manager
	if mon.conf.UseACME {

		m = &autocert.Manager{
			Email:  "werner@tu-berlin.de",
			Prompt: autocert.AcceptTOS,
			HostPolicy: func(ctx context.Context, host string) error {
				//TODO: add sensible host model
				return nil
			},
			Cache: autocert.DirCache(".certs"),
		}

		httpsServer := &http.Server{
			Addr:      ":443",
			Handler:   http.HandlerFunc(mon.serve),
			TLSConfig: &tls.Config{GetCertificate: m.GetCertificate},
		}

		go func() {

			err := httpsServer.ListenAndServeTLS("", "")
			if err != nil {
				log.Fatalf("httpsSrv.ListendAndServeTLS() failed with %s", err)
			}
		}()
	} else if mon.conf.UseSelfSigned {
		cert := fmt.Sprintf("%scert.pem", mon.conf.configDir)
		key := fmt.Sprintf("%skey.pem", mon.conf.configDir)

		err := httpscerts.Check(cert, key)
		if err != nil {
			log.Info("could not load self signed keys - generationg some")
			err = httpscerts.Generate(cert, key, "127.0.0.1:443")
			if err != nil {
				log.Fatal("Error: Couldn't create https certs.")
			}
		}
		httpsServer := &http.Server{
			Addr:    ":443",
			Handler: http.HandlerFunc(mon.serve),
		}
		go func() {

			err := httpsServer.ListenAndServeTLS(cert, key)
			if err != nil {
				log.Fatalf("httpsSrv.ListendAndServeTLS() failed with %s", err)
			}
		}()
	}

	httpServer := &http.Server{
		Addr: ":80",
	}

	if m != nil {
		httpServer.Handler = m.HTTPHandler(http.HandlerFunc(mon.serve))
	} else {
		httpServer.Handler = http.HandlerFunc(mon.serve)
	}

	log.Info("request-monitor ready")
	httpServer.ListenAndServe()

}