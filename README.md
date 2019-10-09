# go-tcp-metrics-proxy

A simple TCP proxy that exposes network telemetry metrics via Prometheus.

See [exposed telemetry metrics](#telemetry-metrics-exposed) for a list of metrics exposed by the proxy.

## Installation

```bash
go get github.com/austingebauer/go-tcp-metrics-proxy
cd go-tcp-metrics-proxy
make
```

An executable named `mproxy` will be placed into the bin 
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
./bin/mproxy -listen="127.0.0.1:3000" -target="127.0.0.1:3001" -metrics="127.0.0.1:3002"
```

### 3. Connect to the TCP proxy

```bash
nc 127.0.0.1 3000
```

### 4. View Prometheus Telemetry Metrics

```bash
curl http://localhost:3000/metrics
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
 
### 7. Send SIGTERM to the proxy

Observe that the proxy will gracefully terminate when a SIGTERM has been received by draining all 
existing connections until they're finished.

### 8. Send SIGTERM to TCP client or server

Observe that the proxy will finally exit after either the client or server closes it's end of the connection. 

## Telemetry Metrics Exposed

The following is a list of telemetry metrics exposed by the proxy in 
[prometheus text-based format](https://github.com/prometheus/docs/blob/master/content/docs/instrumenting/exposition_formats.md#text-based-format)

TODO
