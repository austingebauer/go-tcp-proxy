// package proxy provides a TCP proxy which exposes telemetry metrics via prometheus instrumentation.
package proxy

import (
	"context"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	networkType = "tcp4"
)

var (
	id                 = uuid.New().String()
	inboundConnCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "inbound_connection_count",
			Help: "The total number of inbound connections established",
		},
		[]string{"id"},
	)
	outboundConnCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "outbound_connection_count",
			Help: "The total number of outbound connections established",
		},
		[]string{"id"},
	)
	inboundBytesCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "inbound_bytes_count",
			Help: "The total number of bytes sent and received on inbound connections",
		},
		[]string{"id"},
	)
	outboundBytesCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "outbound_bytes_count",
			Help: "The total number of bytes sent and received on outbound connections",
		},
		[]string{"id"},
	)
	activeInboundConnCount int64 = 0
	activeInboundConnGauge       = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_inbound_connections",
			Help: "The number of currently active inbound connections",
		},
		[]string{"id"},
	)
	activeOutboundConnCount int64 = 0
	activeOutboundConnGauge       = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_outbound_connections",
			Help: "The number of currently active outbound connections",
		},
		[]string{"id"},
	)
	outboundConnTimeout = 10 * time.Second
)

func init() {
	prometheus.MustRegister(inboundConnCounter)
	prometheus.MustRegister(outboundConnCounter)
	prometheus.MustRegister(inboundBytesCounter)
	prometheus.MustRegister(outboundBytesCounter)
	prometheus.MustRegister(activeInboundConnGauge)
	prometheus.MustRegister(activeOutboundConnGauge)
}

// proxy is a TCP proxy which exposes telemetry metrics via prometheus instrumentation.
type proxy struct {
	config        config
	metricsServer *http.Server
	tcpListener   net.Listener
	tcpDialer     *net.Dialer
	doneCh        chan<- struct{}
}

// NewProxy returns a new proxy having the passed configuration.
// The passed done channel will be closed when the proxy has completed shutting down.
func NewProxy(config config, doneCh chan<- struct{}) *proxy {
	return &proxy{
		config: config,
		doneCh: doneCh,
	}
}

// Start start the proxy by listening on the configured address for TCP connections.
func (p *proxy) Start() error {
	log.Println("starting the TCP proxy")

	// Parse the configuration
	err := p.config.parse()
	if err != nil {
		return err
	}

	// Set up the proxy
	err = p.setup()
	if err != nil {
		return err
	}

	errorCh := make(chan error, 1)

	// Start the prometheus metrics server
	go p.startMetricsServer(errorCh)

	// Start accepting connections on the TCP listener
	go p.startTCPListener(errorCh)

	// Block until an error is received
	err = <-errorCh
	return err
}

// setup sets up the proxy in order to begin accepting connections.
func (p *proxy) setup() error {
	// Set up the metrics server, listener, and dialer
	metricsServer := p.setupMetricsServer()
	tcpDialer := p.setupTCPDialer()
	tcpListener, err := p.setupTCPListener()
	if err != nil {
		return err
	}

	// Assign them to the proxy
	p.metricsServer = metricsServer
	p.tcpDialer = &tcpDialer
	p.tcpListener = tcpListener

	return nil
}

// StopForceful stops the proxy forcefully by severing all TCP connections.
func (p *proxy) StopForceful() {
	log.Println("forcefully stopping the TCP proxy")

	err := p.stopMetricsServerForceful()
	if err != nil {
		log.Printf("error occurred shutting down prometheus metrics server: %v", err)
	}

	err = p.stopTCPListenerForceful()
	if err != nil {
		log.Printf("error occurred shutting down TCP listener: %v", err)
	}

	close(p.doneCh)
}

// StopGraceful stops the proxy gracefully by bleeding off all TCP connections.
// The proxy will continue to copy bytes for existing TCP connections.
// The proxy will not accept any new TCP connections.
func (p *proxy) StopGraceful() {
	log.Println("gracefully stopping the TCP proxy")

	err := p.stopMetricsServerGraceful()
	if err != nil {
		log.Printf("error occurred gracefully shutting down prometheus metrics server: %v", err)
	}

	err = p.stopTCPListenerGraceful()
	if err != nil {
		log.Printf("error occurred gracefully shutting down TCP listener: %v", err)
	}

	close(p.doneCh)
}

// setupMetricsServer sets up the prometheus metrics server.
func (p *proxy) setupMetricsServer() *http.Server {
	srv := http.Server{
		Addr: p.config.metricsAddress,
	}
	srv.Handler = promhttp.Handler()
	return &srv
}

// startMetricsServer starts the prometheus metrics server.
func (p *proxy) startMetricsServer(errorCh chan<- error) {
	log.Println("started: prometheus metrics server")

	err := p.metricsServer.ListenAndServe()
	if err != http.ErrServerClosed {
		// Error starting or closing listener
		errorCh <- err
	}
}

// stopMetricsServerForceful forcefully stops the prometheus metrics server.
func (p *proxy) stopMetricsServerForceful() error {
	return p.metricsServer.Close()
}

// stopMetricsServerGraceful gracefully stops the prometheus metrics server.
func (p *proxy) stopMetricsServerGraceful() error {
	return p.metricsServer.Shutdown(context.Background())
}

// setupTCPDialer sets up the outbound TCP dialer.
func (p *proxy) setupTCPDialer() net.Dialer {
	return net.Dialer{
		Timeout: time.Minute,
	}
}

// setupTCPListener sets up the incoming TCP listener.
func (p *proxy) setupTCPListener() (net.Listener, error) {
	return net.Listen(networkType, p.config.listenAddress)
}

// startTCPListener starts the TCP listener so that it can accept new connections.
func (p *proxy) startTCPListener(errorCh chan<- error) {
	log.Println("started: TCP connection listener")

	for {
		conn, err := p.tcpListener.Accept()
		if err != nil {
			errorCh <- err
			return
		}

		// update inbound metrics
		inboundConnCounter.WithLabelValues(id).Inc()
		atomic.AddInt64(&activeInboundConnCount, 1)
		activeInboundConnGauge.WithLabelValues(id).Inc()

		go p.handleTCPConnection(conn, errorCh)
	}
}

// stopTCPListenerForceful stops the TCP listener forcefully
// by immediately severing existing connections.
func (p *proxy) stopTCPListenerForceful() error {
	return p.tcpListener.Close()
}

// stopTCPListenerGraceful stops the TCP listener gracefully by bleeding
// all current connections and not accepting any new connections.
func (p *proxy) stopTCPListenerGraceful() error {
	for activeInboundConnCount != 0 && activeOutboundConnCount != 0 {
		log.Printf("draining %d connections", activeInboundConnCount+activeOutboundConnCount)
		time.Sleep(time.Second * 5)
	}

	return nil
}

func (p *proxy) handleTCPConnection(inboundConn net.Conn, errorCh chan<- error) {
	ctx, cancel := context.WithTimeout(context.Background(), outboundConnTimeout)
	defer cancel()

	// Dial for an outbound connection
	outboundConn, err := p.tcpDialer.DialContext(ctx, networkType, p.config.targetAddress)
	if err != nil {
		// Could not establish outbound connection, so close inbound connection
		err := inboundConn.Close()
		if err != nil {
			// Failure to close inbound connection and dial for outbound connection
			// Communicate the error for a fatal exit of the program.
			errorCh <- err
			return
		}

		// Inbound connection has been closed, so decrement active inbound gauge
		atomic.AddInt64(&activeInboundConnCount, -1)
		activeInboundConnGauge.WithLabelValues(id).Dec()

		// Failing to dial does not kill the process, so just log the error and return
		log.Println(err)
		return
	}

	// Outbound connection established, so increment active outbound gauge
	outboundConnCounter.WithLabelValues(id).Inc()
	atomic.AddInt64(&activeOutboundConnCount, 1)
	activeOutboundConnGauge.WithLabelValues(id).Inc()

	// Channels to communicate amount of bytes copied
	// between inbound and outbound connections
	inboundBytesCh := make(chan int64, 1)
	outboundBytesCh := make(chan int64, 1)

	log.Printf("connection started: client=%v destination=%v",
		inboundConn.RemoteAddr().String(),
		outboundConn.RemoteAddr().String())
	start := time.Now()

	// Block until amount of bytes copied is communicated over each channel
	go p.copy(outboundConn.(*net.TCPConn), inboundConn.(*net.TCPConn), inboundBytesCh)
	go p.copy(inboundConn.(*net.TCPConn), outboundConn.(*net.TCPConn), outboundBytesCh)
	inboundBytesCopied, outboundBytesCopied := <-inboundBytesCh, <-outboundBytesCh

	elapsed := time.Now().Sub(start)
	log.Printf("connection ended: client=%v destination=%v duration=%v bytes_copied=%d",
		inboundConn.RemoteAddr().String(),
		outboundConn.RemoteAddr().String(),
		elapsed.String(),
		inboundBytesCopied+outboundBytesCopied)

	// Connection proxying complete, so update all metrics
	inboundBytesCounter.WithLabelValues(id).Add(float64(inboundBytesCopied))
	outboundBytesCounter.WithLabelValues(id).Add(float64(outboundBytesCopied))
	atomic.AddInt64(&activeInboundConnCount, -1)
	activeInboundConnGauge.WithLabelValues(id).Dec()
	atomic.AddInt64(&activeOutboundConnCount, -1)
	activeOutboundConnGauge.WithLabelValues(id).Dec()
}

// copy copies bytes from the passed reader TCP connection to the passed writer
// TCP connection until either EOF is reached on src or an error occurs.
func (p *proxy) copy(writer *net.TCPConn, reader *net.TCPConn, byteCountCh chan<- int64) {
	bytesCopied, err := io.Copy(writer, reader)
	if err != nil {
		log.Println(err)
	}

	err = writer.CloseWrite()
	if err != nil {
		log.Println(err)
	}

	err = reader.CloseRead()
	if err != nil {
		log.Println(err)
	}

	byteCountCh <- bytesCopied
}
