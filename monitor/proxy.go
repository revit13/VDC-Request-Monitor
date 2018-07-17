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
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	log "github.com/sirupsen/logrus"
)

func (mon *RequestMonitor) serve(w http.ResponseWriter, req *http.Request) {
	var requestID = mon.generateRequestID(req.RemoteAddr)

	//read payload
	if mon.conf.ForwardTraffic {
		body, err := ioutil.ReadAll(req.Body)

		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}

		//enact proxy request
		req.ContentLength = int64(len(body))
		req.Body = ioutil.NopCloser(bytes.NewReader(body))

		exchange := exchangeMessage{
			RequestBody:   string(body),
			RequestHeader: req.Header,
		}

		mon.forward(requestID, exchange)
	}
	method := req.URL.Path
	req.URL = mon.conf.endpointURL

	//inject tracing header
	if mon.conf.Opentracing {
		opentracing.GlobalTracer().Inject(
			opentracing.StartSpan("VDC-Request").Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(req.Header),
		)
	}

	//inject looging header
	req.Header.Set("X-DITAS-RequestID", requestID)

	//forward the request
	start := time.Now()
	mon.oxy.ServeHTTP(w, req)
	end := time.Now().Sub(start)

	//report all logging information
	meter := MeterMessage{
		Client:        req.RemoteAddr,
		Method:        method,
		Kind:          req.Method,
		RequestLenght: req.ContentLength,
		RequestTime:   end,
	}

	mon.push(requestID, meter)
}

func (mon *RequestMonitor) responseInterceptor(resp *http.Response) error {
	//extract requestID
	requestID := resp.Request.Header.Get("X-DITAS-RequestID")
	if requestID == "" {
		requestID = mon.generateRequestID(resp.Request.RemoteAddr)
	}

	meter := MeterMessage{
		RequestID:      requestID,
		ResponseCode:   resp.StatusCode,
		ResponseLength: resp.ContentLength,
	}
	mon.push(requestID, meter)

	if !mon.conf.ForwardTraffic {
		return nil
	}

	//read the body and reset the reader (otherwise it will not be availible to the client)
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Error reading body: %v", err)
		return err
	}

	log.Infof("%s", string(body))

	resp.ContentLength = int64(len(body))
	resp.Body = ioutil.NopCloser(bytes.NewReader(body))

	//report logging information
	exchange := exchangeMessage{
		ResponseBody:   string(body),
		ResponseHeader: resp.Header,
	}
	mon.forward(requestID, exchange)
	return nil
}
