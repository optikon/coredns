package central

import (
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"

	"github.com/mholt/caddy"
)

// Registers plugin upon package import.
func init() {
	caddy.RegisterPlugin("optikon-central", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

// Specifies everything to be run/configured before serving DNS queries.
func setup(c *caddy.Controller) error {

	// Parse the plugin arguments.
	oc, err := parseOptikonCentral(c)
	if err != nil {
		return plugin.Error("optikon-central", err)
	}

	// Add the plugin handler to the dnsserver.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return oc
	})

	// Start Go routine for listening to edge clusters for Table updates.
	go oc.listenForTableUpdates()

	return nil
}

// Parse the Corefile token.
func parseOptikonCentral(c *caddy.Controller) (*OptikonCentral, error) {

	// Initialize a new OptikonCentral struct.
	oc := New()

	// Skip the 'optikon-central' token.
	c.Next()

	// If there are any other arguments, throw an error.
	if c.NextArg() {
		return oc, c.ArgErr()
	}

	return oc, nil
}
