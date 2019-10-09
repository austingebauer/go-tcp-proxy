package main

import (
	"flag"
	"github.com/austingebauer/go-tcp-metrics-proxy/proxy"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	listenAddress string
	targetAddress string
	metricAddress string
)

func init() {
	flag.StringVar(&listenAddress, "listen", "127.0.0.1:3000",
		"IP address and port number that the proxy will listen on")
	flag.StringVar(&targetAddress, "target", "127.0.0.1:3001",
		"IP address and port number that the proxy will forward to")
	flag.StringVar(&metricAddress, "metrics", "127.0.0.1:3002",
		"IP address and port number to expose prometheus metrics on")
}

func main() {
	// Parse flags and assign to configuration
	flag.Parse()
	config := proxy.NewConfig(listenAddress, targetAddress, metricAddress)

	// Set up channels and signal handling
	errorCh := make(chan error)
	doneCh := make(chan struct{})
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// Configure and run the proxy
	p := proxy.NewProxy(config, doneCh)
	go func() {
		errorCh <- p.Start()
	}()

	var finalError error

	// Block until an error or signal is received
	select {
	case sig := <-signalCh:
		log.Printf("received signal: %v\n", sig)

		// Stop gracefully for SIGTERM and SIGINT
		p.StopGraceful()
	case err := <-errorCh:
		finalError = err

		// Stop forcefully for errors
		p.StopForceful()
	}

	// Block until the done channel has been closed by the proxy
	<-doneCh

	// If the proxy stopped due to an error, then log fatally
	if finalError != nil {
		log.Fatal(finalError)
	}

	// Otherwise, the proxy stopped due to a signal, so exit 0
	log.Println("exit: 0")
	os.Exit(0)
}
