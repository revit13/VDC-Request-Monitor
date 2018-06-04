package monitor

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	opentracing "github.com/opentracing/opentracing-go"

	log "github.com/sirupsen/logrus"
)

func (mon *RequestMonitor) serve(w http.ResponseWriter, req *http.Request) {

	//read payload

	body, err := ioutil.ReadAll(req.Body)

	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	//enact proxy request
	req.URL = mon.conf.EndpointURL
	req.ContentLength = int64(len(body))
	req.Body = ioutil.NopCloser(bytes.NewReader(body))

	//inject tracing header
	if mon.conf.Opentracing {
		opentracing.GlobalTracer().Inject(
			opentracing.StartSpan("VDC-Request").Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(req.Header),
		)
	}

	//inject looging header
	var requestID = mon.generateRequestID(req.RemoteAddr)
	req.Header.Set("X-DITAS-RequestID", requestID)

	//forward the request
	start := time.Now()
	mon.oxy.ServeHTTP(w, req)
	end := time.Now().Sub(start)

	//report all logging information
	meter := meterMessage{
		Client:      req.RemoteAddr,
		Method:      req.URL.String(),
		RequestTime: end,
	}

	mon.push(requestID, meter)

	exchange := exchangeMessage{
		RequestBody:   string(body),
		RequestHeader: req.Header,
	}
	mon.forward(requestID, exchange)

}

func (mon *RequestMonitor) responseInterceptor(resp *http.Response) error {

	//read the body and reset the reader (otherwise it will not be availible to the client)
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Error reading body: %v", err)
		return err
	}

	log.Infof("%s", string(body))

	resp.ContentLength = int64(len(body))
	resp.Body = ioutil.NopCloser(bytes.NewReader(body))

	//extract requestID
	requestID := resp.Request.Header.Get("X-DITAS-RequestId")
	if requestID == "" {
		requestID = mon.generateRequestID(resp.Request.RemoteAddr)
	}
	//report logging information
	exchange := exchangeMessage{
		ResponseBody:   string(body),
		ResponseHeader: resp.Header,
	}
	mon.forward(requestID, exchange)
	return nil
}
