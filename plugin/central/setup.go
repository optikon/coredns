package central

import (
	"fmt"
	"strconv"
	"time"

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
	fmt.Printf("Parsed OptikonCentral with Parameters: IP=%s, Lon=%f, Lat=%f, svcReadInterval=%v\n", oc.ip, oc.lon, oc.lat, oc.svcReadInterval)

	// Add the plugin handler to the dnsserver.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		oc.clientset, err = RegisterKubernetesClient()
		oc.Next = next
		return oc
	})

	c.OnStartup(func() error {
		return oc.OnStartup()
	})

	c.OnShutdown(func() error {
		return oc.OnShutdown()
	})

	return nil
}

// OnStartup starts starts reading services and listening for edge pushes.
func (oc *OptikonCentral) OnStartup() (err error) {
	oc.startReadingServices()
	oc.startListeningForTableUpdates()
	return nil
}

// OnShutdown stops all async processes.
func (oc *OptikonCentral) OnShutdown() error {
	oc.startReadingServices()
	oc.stopListeningForTableUpdates()
	return nil
}

// Parse the Corefile token.
func parseOptikonCentral(c *caddy.Controller) (*OptikonCentral, error) {

	// Initialize a new OptikonCentral struct.
	oc := New()

	// Skip the 'optikon-central' token.
	c.Next()

	// Parse my IP address.
	if !c.Args(&oc.ip) {
		return oc, c.ArgErr()
	}

	// Parse the edge cluster's longitude value.
	var lon string
	if !c.Args(&lon) {
		return oc, c.ArgErr()
	}
	parsedLon, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		return oc, err
	}
	oc.lon = parsedLon

	// Parse the latitude value.
	var lat string
	if !c.Args(&lat) {
		return oc, c.ArgErr()
	}
	parsedLat, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		return oc, err
	}
	oc.lat = parsedLat

	// Parse the service read interval.
	var svcReadIntervalSecsString string
	if !c.Args(&svcReadIntervalSecsString) {
		return oc, c.ArgErr()
	}
	svcReadIntervalSecs, err := strconv.Atoi(svcReadIntervalSecsString)
	if err != nil {
		return oc, err
	}
	oc.svcReadInterval = time.Duration(svcReadIntervalSecs) * time.Second

	// If there are any other arguments, throw an error.
	if c.NextArg() {
		return oc, c.ArgErr()
	}

	return oc, nil
}
