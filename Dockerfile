FROM golang:1.10.1 AS build
ENV SOURCEDIR=/go/src/github.com/DITAS-Project/VDC-Request-Monitor
RUN mkdir -p ${SOURCEDIR}
WORKDIR ${SOURCEDIR}
COPY . .
RUN rm -rf vendor/ && go get -u github.com/golang/dep/cmd/dep && dep ensure
#Patching opentracing
#RUN patch vendor/github.com/openzipkin/zipkin-go-opentracing/thrift/gen-go/scribe/scribe.go scribe.patch
RUN CGO_ENABLED=0 GOOS=linux go build -a --installsuffix cgo --ldflags="-w -s -X main.Build=$(git rev-parse --short HEAD)" -o request-monitor

FROM alpine:3.4
COPY --from=build /go/src/github.com/DITAS-Project/VDC-Request-Monitor/request-monitor request-monitor
#ADD .config/monitor.json.example .config/monitor.json
EXPOSE 80
EXPOSE 443
CMD [ "./request-monitor" ]