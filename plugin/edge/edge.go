package edge

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"wwwin-github.cisco.com/edge/optikon-dns/plugin/central"

	"github.com/miekg/dns"
	ot "github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
)

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

	lon      float64
	lat      float64
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

// ServeDNS implements plugin.Handler.
func (oe *OptikonEdge) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

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

		// Assert an additional entry for the table exists.
		if len(ret.Extra) == 0 {
			return dns.RcodeServerFailure, errTableParseFailure
		}

		// Extract the Table from the response.
		tabRR := ret.Extra[0]
		tabSubmatches := tableRegex.FindStringSubmatch(tabRR.String())
		if len(tabSubmatches) < 2 {
			return dns.RcodeServerFailure, errTableParseFailure
		}
		tabStr, err := strconv.Unquote(fmt.Sprintf("\"%s\"", tabSubmatches[1]))
		if err != nil {
			return dns.RcodeServerFailure, errTableParseFailure
		}
		var tab central.Table
		if err := json.Unmarshal([]byte(tabStr), &tab); err != nil {
			return dns.RcodeServerFailure, errTableParseFailure
		}

		// Remove the Table entry from the return message.
		ret.Extra = ret.Extra[1:]

		// Parse the target domain out of the request (NOTE: This will always have
		// a trailing dot.)
		targetDomain := state.Name()

		// Determine if there is an entry for the DNS name we're looking for.
		edgeSites, found := tab[targetDomain[:(len(targetDomain)-1)]]
		if !found || len(edgeSites) == 0 {
			w.WriteMsg(ret)
			return 0, nil
		}

		// Compute the distance to the first edge site.
		closest := edgeSites[0].IP
		minDist := Distance(oe.lat, oe.lon, edgeSites[0].Lat, edgeSites[0].Lon)
		for _, edgeSite := range edgeSites {
			dist := Distance(oe.lat, oe.lon, edgeSite.Lat, edgeSite.Lon)
			if dist < minDist {
				minDist = dist
				closest = edgeSite.IP
			}
		}

		// Write the closest cluster IP as a DNS record.
		var rr dns.RR
		switch state.Family() {
		case 1:
			rr = new(dns.A)
			rr.(*dns.A).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass()}
			rr.(*dns.A).A = net.ParseIP(closest).To4()
		case 2:
			rr = new(dns.AAAA)
			rr.(*dns.AAAA).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeAAAA, Class: state.QClass()}
			rr.(*dns.AAAA).AAAA = net.ParseIP(closest)
		}
		ret.Answer = []dns.RR{rr}

		// Write the response message.
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
	errInvalidDomain         = errors.New("invalid domain for forward")
	errNoHealthy             = errors.New("no healthy proxies")
	errNoOptikonEdge         = errors.New("no optikon-edge defined")
	errTableParseFailure     = errors.New("unable to parse Table returned from central")
	errFindingClosestCluster = errors.New("unable to compute closest edge cluster")
)

// policy tells forward what policy for selecting upstream it uses.
type policy int

const (
	randomPolicy policy = iota
	roundRobinPolicy
)

var (
	tableRegex = regexp.MustCompile("^.*\t0\tIN\tTXT\t\"({.*})\"$")
)
