# Readme
The request-monitor consists of a reverse-proxy that can log all request metadata elasticsearch and also is able to observe all incoming and outgoing requests for further analysis. It also contains an opentracing instrumentation if that is needed.

## Usage
To configure the proxy have a lock at the .config/monitor.json.example. That file contains all the configuration needed to run the proxy.
The proxy is able to automatically get a SSL certificate from lets encrypted (only if you use a domain) otherwise it can use self signed certificates.

To run the proxy use ```dep ensure && go build && ./VDC-Request-Monitor```
