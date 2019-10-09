package proxy

import (
	"net"
)

// config is the configuration required to run a proxy
type config struct {
	listenAddress  string
	listenHost     string
	listenPort     string
	targetAddress  string
	targetHost     string
	targetPort     string
	metricsAddress string
	metricsHost    string
	metricsPort    string
}

// NewConfig returns a new
func NewConfig(listenAddress, targetAddress, metricsAddress string) config {
	return config{
		listenAddress:  listenAddress,
		targetAddress:  targetAddress,
		metricsAddress: metricsAddress,
	}
}

// parse parses this config.
// Returns an error if its values are not parsable.
func (c *config) parse() error {
	var err error

	c.listenHost, c.listenPort, err = net.SplitHostPort(c.listenAddress)
	if err != nil {
		return err
	}

	c.targetHost, c.targetPort, err = net.SplitHostPort(c.targetAddress)
	if err != nil {
		return err
	}

	c.metricsHost, c.metricsPort, err = net.SplitHostPort(c.metricsAddress)
	if err != nil {
		return err
	}

	return nil
}
