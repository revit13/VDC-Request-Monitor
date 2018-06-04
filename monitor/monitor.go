package monitor

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"

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

	exporter, err := newExchangeReporter(configuration.ExchangeReporterURL, mng.exchangeQueue)
	if err != nil {
		log.Errorf("Failed to init exchange reporter %+v", err)
		return nil, err
	}
	mng.exporter = exporter

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

//Listen will start all worker threads and wait for incoming requests
func (mon *RequestMonitor) Listen() {
	s := &http.Server{
		Addr:    ":80",
		Handler: http.HandlerFunc(mon.serve),
	}

	//start parallel reporter threads
	mon.reporter.Start()
	mon.exporter.Start()

	defer mon.reporter.Stop()
	defer mon.exporter.Stop()

	log.Info("request-monitor ready")
	s.ListenAndServe()

}

func (mon *RequestMonitor) push(requestID string, message meterMessage) {
	message.RequestID = requestID
	message.Timestamp = time.Now()
	mon.monitorQueue <- message
}

func (mon *RequestMonitor) forward(requestID string, message exchangeMessage) {
	message.RequestID = requestID
	message.Timestamp = time.Now()
	mon.exchangeQueue <- message
}
