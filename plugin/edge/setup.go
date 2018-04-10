package edge

import (
	"errors"
	"net"
	"strconv"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"

	"github.com/mholt/caddy"
)

// Registers plugin upon package import.
func init() {
	caddy.RegisterPlugin("optikon-edge", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

// Specifies everything to be run/configured before serving DNS queries.
func setup(c *caddy.Controller) error {

	// Parse the plugin arguments.
	oe, err := parseOptikonEdge(c)
	if err != nil {
		return plugin.Error("optikon-edge", err)
	}

	// Add the plugin handler to the dnsserver.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return oe
	})

	// TODO: START ROUTINE TO READ RUNNING SERVICES EVERY MINUTE AND PUSH UP TO
	// CENTRAL + UPDATE LOCAL `services` LIST (?)

	return nil
}

// Parse the Corefile token.
func parseOptikonEdge(c *caddy.Controller) (*OptikonEdge, error) {

	// Initialize a new OptikonEdge struct.
	oe := New()

	// Skip the 'optikon-edge' token.
	c.Next()

	// Assert that there are enough args left.
	args := c.RemainingArgs()
	if len(args) != 3 {
		return oe, errors.New("incorrect number of plugin arguments (expecting 3)")
	}

	// Parse the central cluster IP.
	parsedIP := net.ParseIP(args[0])
	if parsedIP == nil {
		return oe, errors.New("invalid central cluster IP address")
	}
	oe.SetCentralIP(parsedIP.String())

	// Parse the arguments as lon-lat coordinates.
	lon, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return oe, err
	}
	oe.SetLon(lon)
	lat, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		return oe, err
	}
	oe.SetLat(lat)

	return oe, nil
}
