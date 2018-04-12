package edge

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
	ot "github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
)

// Site is a wrapper around all information needed about edge sites serving
// content.
type Site struct {
	IP  string  `json:"ip"`
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

// Coords is a 2-tuple of longitude and latitude values.
type Coords [2]float64

// OptikonEdge represents a plugin instance that can proxy requests to another (DNS) server. It has a list
// of proxies each representing one upstream proxy.
type OptikonEdge struct {
	proxies    []*Proxy
	p          Policy
	hcInterval time.Duration

	from    string
	ignored []string

	tlsConfig     *tls.Config
	tlsServerName string
	maxfails      uint32
	expire        time.Duration

	forceTCP bool // also here for testing

	Next plugin.Handler

	coords   Coords
	services []string
}

// New returns a new OptikonEdge.
func New() *OptikonEdge {
	oe := &OptikonEdge{maxfails: 2, tlsConfig: new(tls.Config), expire: defaultExpire, p: new(random), from: ".", hcInterval: hcDuration}
	return oe
}

// SetProxy appends p to the proxy list and starts healthchecking.
func (oe *OptikonEdge) SetProxy(p *Proxy) {
	oe.proxies = append(oe.proxies, p)
	p.start(oe.hcInterval)
}

// Len returns the number of configured proxies.
func (oe *OptikonEdge) Len() int { return len(oe.proxies) }

// Name implements plugin.Handler.
func (oe *OptikonEdge) Name() string { return "optikon-edge" }

// SetLon sets the edge site longitude.
func (oe *OptikonEdge) SetLon(v float64) { oe.coords[0] = v }

// SetLat sets the edge site latitude.
func (oe *OptikonEdge) SetLat(v float64) { oe.coords[1] = v }

// ServeDNS implements plugin.Handler.
func (oe *OptikonEdge) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	fmt.Println("REQUEST:", r.String())

	state := request.Request{W: w, Req: r}
	if !oe.match(state) {
		return plugin.NextOrFailure(oe.Name(), oe.Next, ctx, w, r)
	}

	fails := 0
	var span, child ot.Span
	var upstreamErr error
	span = ot.SpanFromContext(ctx)

	for _, proxy := range oe.list() {
		if proxy.Down(oe.maxfails) {
			fails++
			if fails < len(oe.proxies) {
				continue
			}
			// All upstream proxies are dead, assume healtcheck is completely broken and randomly
			// select an upstream to connect to.
			r := new(random)
			proxy = r.List(oe.proxies)[0]

			HealthcheckBrokenCount.Add(1)
		}

		if span != nil {
			child = span.Tracer().StartSpan("connect", ot.ChildOf(span.Context()))
			ctx = ot.ContextWithSpan(ctx, child)
		}

		var (
			ret *dns.Msg
			err error
		)
		stop := false
		for {
			ret, err = proxy.connect(ctx, state, oe.forceTCP, true)
			if err != nil && err == io.EOF && !stop { // Remote side closed conn, can only happen with TCP.
				stop = true
				continue
			}
			break
		}

		if child != nil {
			child.Finish()
		}

		ret, err = truncated(ret, err)
		upstreamErr = err

		if err != nil {
			// Kick off health check to see if *our* upstream is broken.
			if oe.maxfails != 0 {
				proxy.Healthcheck()
			}

			if fails < len(oe.proxies) {
				continue
			}
			break
		}

		// Check if the reply is correct; if not return FormErr.
		if !state.Match(ret) {
			formerr := state.ErrorMessage(dns.RcodeFormatError)
			w.WriteMsg(formerr)
			return 0, nil
		}

		ret.Compress = true
		// When using force_tcp the upstream can send a message that is too big for
		// the udp buffer, hence we need to truncate the message to at least make it
		// fit the udp buffer.
		ret, _ = state.Scrub(ret)

		fmt.Println("PROXY REPLY MESSAGE:", ret.String())

		w.WriteMsg(ret)

		return 0, nil
	}

	if upstreamErr != nil {
		return dns.RcodeServerFailure, upstreamErr
	}

	return dns.RcodeServerFailure, errNoHealthy
}

func (oe *OptikonEdge) match(state request.Request) bool {
	from := oe.from

	if !plugin.Name(from).Matches(state.Name()) || !oe.isAllowedDomain(state.Name()) {
		return false
	}

	return true
}

func (oe *OptikonEdge) isAllowedDomain(name string) bool {
	if dns.Name(name) == dns.Name(oe.from) {
		return true
	}

	for _, ignore := range oe.ignored {
		if plugin.Name(ignore).Matches(name) {
			return false
		}
	}
	return true
}

// List returns a set of proxies to be used for this client depending on the policy in oe.
func (oe *OptikonEdge) list() []*Proxy { return oe.p.List(oe.proxies) }

var (
	errInvalidDomain = errors.New("invalid domain for forward")
	errNoHealthy     = errors.New("no healthy proxies")
	errNoOptikonEdge = errors.New("no optikon-edge defined")
)

// policy tells forward what policy for selecting upstream it uses.
type policy int

const (
	randomPolicy policy = iota
	roundRobinPolicy
)
