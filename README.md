# go-tcp-proxy

A simple TCP proxy that exposes network telemetry metrics via Prometheus instrumentation.

See [exposed telemetry metrics](#telemetry-metrics-exposed) for a list of metrics exposed by the proxy.

## Installation

```bash
go get github.com/austingebauer/go-tcp-metrics-proxy
cd go-tcp-metrics-proxy
make
```

An executable named `proxy` will be placed into the bin 
directory after the make target completes.

## Usage

go-tcp-metrics-proxy is configured using command line arguments.

The example below shows the usage of go-tcp-metrics-proxy using netcat as a client and server.

### 1. Start listening for TCP connections

```bash
nc -k -l 127.0.0.1 3001
```

### 2. Start the TCP proxy

```bash
./bin/proxy -listen="127.0.0.1:3000" -target="127.0.0.1:3001" -metrics="127.0.0.1:3002"
```

### 3. Connect to the TCP proxy

```bash
nc 127.0.0.1 3000
```

### 4. View Prometheus Telemetry Metrics

```bash
curl http://localhost:3002/metrics
```

### 5. Send bytes from client to server

In the window running the netcat TCP client program, type the following followed by a carriage return:
```bash
Hello, server!
```

Observe that the message has been arrived on the netcat TCP server as terminal output.

Sending bytes from server to client works the same way, so give that a try as well.

### 6. View Prometheus Telemetry Metrics.. Again! 

 ```bash
 curl http://localhost:3000/metrics
 ```

Observe that metrics related to TCP connections handled by the proxy have been updated.
 
### 7. Send SIGTERM to the proxy

Observe that the proxy will gracefully terminate when a SIGTERM has been received by draining all 
existing connections until they're finished.

### 8. Send SIGTERM to TCP client or server

Observe that the proxy will finally exit after either the client or server closes its end of the connection. 

## Telemetry Metrics Exposed

The following is a list of telemetry metrics exposed by the proxy in 
[prometheus text-based format](https://github.com/prometheus/docs/blob/master/content/docs/instrumenting/exposition_formats.md#text-based-format)

Metrics related to Go have been omitted from the `/metrics` results below in order to showcase 
the TCP proxy related metrics.

```
# HELP active_inbound_connections The number of currently active inbound connections
# TYPE active_inbound_connections gauge
active_inbound_connections{id="75fc83c4-2109-4757-8660-896c170303c3"} 0
# HELP active_outbound_connections The number of currently active outbound connections
# TYPE active_outbound_connections gauge
active_outbound_connections{id="75fc83c4-2109-4757-8660-896c170303c3"} 0
# HELP inbound_bytes_count The total number of bytes sent and received on inbound connections
# TYPE inbound_bytes_count counter
inbound_bytes_count{id="75fc83c4-2109-4757-8660-896c170303c3"} 15
# HELP inbound_connection_count The total number of inbound connections established
# TYPE inbound_connection_count counter
inbound_connection_count{id="75fc83c4-2109-4757-8660-896c170303c3"} 1
# HELP outbound_bytes_count The total number of bytes sent and received on outbound connections
# TYPE outbound_bytes_count counter
outbound_bytes_count{id="75fc83c4-2109-4757-8660-896c170303c3"} 15
# HELP outbound_connection_count The total number of outbound connections established
# TYPE outbound_connection_count counter
outbound_connection_count{id="75fc83c4-2109-4757-8660-896c170303c3"} 1
# HELP promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
# TYPE promhttp_metric_handler_requests_in_flight gauge
promhttp_metric_handler_requests_in_flight 1
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 3
promhttp_metric_handler_requests_total{code="500"} 0
promhttp_metric_handler_requests_total{code="503"} 0
```
