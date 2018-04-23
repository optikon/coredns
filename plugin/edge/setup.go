/*
 * Copyright 2018 The CoreDNS Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may
 * not use this file except in compliance with the License. You may obtain
 * a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * NOTE: This software contains code derived from the the Apache-licensed CoreDNS
 * `forward` plugin (https://github.com/coredns/coredns/tree/master/plugin/forward),
 * including various modifications by Cisco Systems, Inc.
 */

package edge

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	pkgtls "github.com/coredns/coredns/plugin/pkg/tls"

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
	if oe.Len() > max {
		return plugin.Error("optikon-edge", fmt.Errorf("more than %d TOs configured: %d", max, oe.Len()))
	}

	// Add the plugin handler to the dnsserver.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		oe.Next = next
		return oe
	})

	// Register Prometheus metrics.
	c.OnStartup(func() error {
		once.Do(func() {
			metrics.MustRegister(c, RequestCount, RcodeCount, RequestDuration, HealthcheckFailureCount, SocketGauge)
		})
		return oe.OnStartup()
	})

	c.OnShutdown(func() error {
		return oe.OnShutdown()
	})

	return nil
}

// OnStartup starts a goroutines for all proxies.
func (oe *OptikonEdge) OnStartup() (err error) {
	for _, p := range oe.proxies {
		p.start(oe.hcInterval)
	}
	return nil
}

// OnShutdown stops all configured proxies.
func (oe *OptikonEdge) OnShutdown() error {
	for _, p := range oe.proxies {
		p.close()
	}
	return nil
}

// Close is a synonym for OnShutdown().
func (oe *OptikonEdge) Close() { oe.OnShutdown() }

// Parse the Corefile tokens associated with this plugin.
func parseOptikonEdge(c *caddy.Controller) (*OptikonEdge, error) {

	// Initialize a new OptikonEdge struct.
	oe := New()

	protocols := map[int]int{}

	i := 0
	for c.Next() {
		if i > 0 {
			return nil, plugin.ErrOnce
		}
		i++

		// Parse the edge cluster's longitude value.
		var lon string
		if !c.Args(&lon) {
			return oe, c.ArgErr()
		}
		parsedLon, err := strconv.ParseFloat(lon, 64)
		if err != nil {
			return oe, err
		}
		oe.lon = parsedLon

		// Parse the latitude value.
		var lat string
		if !c.Args(&lat) {
			return oe, c.ArgErr()
		}
		parsedLat, err := strconv.ParseFloat(lat, 64)
		if err != nil {
			return oe, err
		}
		oe.lat = parsedLat

		if !c.Args(&oe.from) {
			return oe, c.ArgErr()
		}
		oe.from = plugin.Host(oe.from).Normalize()

		to := c.RemainingArgs()
		if len(to) == 0 {
			return oe, c.ArgErr()
		}

		// A bit fiddly, but first check if we've got protocols and if so add them back in when we create the proxies.
		protocols = make(map[int]int)
		for i := range to {
			protocols[i], to[i] = protocol(to[i])
		}

		// If parseHostPortOrFile expands a file with a lot of nameserver our accounting in protocols doesn't make
		// any sense anymore... For now: lets don't care.
		toHosts, err := dnsutil.ParseHostPortOrFile(to...)
		if err != nil {
			return oe, err
		}

		for i, h := range toHosts {
			// Double check the port, if e.g. is 53 and the transport is TLS make it 853.
			// This can be somewhat annoying because you *can't* have TLS on port 53 then.
			switch protocols[i] {
			case TLS:
				h1, p, err := net.SplitHostPort(h)
				if err != nil {
					break
				}

				// This is more of a bug in // dnsutil.ParseHostPortOrFile that defaults to
				// 53 because it doesn't know about the tls:// // and friends (that should be fixed). Hence
				// Fix the port number here, back to what the user intended.
				if p == "53" {
					h = net.JoinHostPort(h1, "853")
				}
			}

			// We can't set tlsConfig here, because we haven't parsed it yet.
			// We set it below at the end of parseBlock, use nil now.
			p := NewProxy(h, nil /* no TLS */)
			oe.proxies = append(oe.proxies, p)
		}

		for c.NextBlock() {
			if err := parseBlock(c, oe); err != nil {
				return oe, err
			}
		}
	}

	if oe.tlsServerName != "" {
		oe.tlsConfig.ServerName = oe.tlsServerName
	}
	for i := range oe.proxies {
		// Only set this for proxies that need it.
		if protocols[i] == TLS {
			oe.proxies[i].SetTLSConfig(oe.tlsConfig)
		}
		oe.proxies[i].SetExpire(oe.expire)
	}
	return oe, nil
}

func parseBlock(c *caddy.Controller, oe *OptikonEdge) error {
	switch c.Val() {
	case "except":
		ignore := c.RemainingArgs()
		if len(ignore) == 0 {
			return c.ArgErr()
		}
		for i := 0; i < len(ignore); i++ {
			ignore[i] = plugin.Host(ignore[i]).Normalize()
		}
		oe.ignored = ignore
	case "max_fails":
		if !c.NextArg() {
			return c.ArgErr()
		}
		n, err := strconv.Atoi(c.Val())
		if err != nil {
			return err
		}
		if n < 0 {
			return fmt.Errorf("max_fails can't be negative: %d", n)
		}
		oe.maxfails = uint32(n)
	case "health_check":
		if !c.NextArg() {
			return c.ArgErr()
		}
		dur, err := time.ParseDuration(c.Val())
		if err != nil {
			return err
		}
		if dur < 0 {
			return fmt.Errorf("health_check can't be negative: %d", dur)
		}
		oe.hcInterval = dur
	case "force_tcp":
		if c.NextArg() {
			return c.ArgErr()
		}
		oe.forceTCP = true
	case "tls":
		args := c.RemainingArgs()
		if len(args) > 3 {
			return c.ArgErr()
		}

		tlsConfig, err := pkgtls.NewTLSConfigFromArgs(args...)
		if err != nil {
			return err
		}
		oe.tlsConfig = tlsConfig
	case "tls_servername":
		if !c.NextArg() {
			return c.ArgErr()
		}
		oe.tlsServerName = c.Val()
	case "expire":
		if !c.NextArg() {
			return c.ArgErr()
		}
		dur, err := time.ParseDuration(c.Val())
		if err != nil {
			return err
		}
		if dur < 0 {
			return fmt.Errorf("expire can't be negative: %s", dur)
		}
		oe.expire = dur
	case "policy":
		if !c.NextArg() {
			return c.ArgErr()
		}
		switch x := c.Val(); x {
		case "random":
			oe.p = &random{}
		case "round_robin":
			oe.p = &roundRobin{}
		default:
			return c.Errf("unknown policy '%s'", x)
		}

	default:
		return c.Errf("unknown property '%s'", c.Val())
	}

	return nil
}

const max = 15 // Maximum number of upstreams.
