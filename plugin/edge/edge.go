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

		fmt.Println("PROXY REPLY:", ret)

		// Assert an additional entry for the table exists.
		if len(ret.Extra) == 0 {
			return 1, errors.New("expected Extra entry to be non-empty")
		}

		// Extract the Table from the response.
		tabRR := ret.Extra[0]
		re := regexp.MustCompile("^.*\t0\tIN\tTXT\t\"({.*})\"$")
		tabSubmatches := re.FindStringSubmatch(tabRR.String())
		fmt.Println("SUBMATCHES:", tabSubmatches)
		if len(tabSubmatches) < 2 {
			return 2, errors.New("unable to parse Table returned from central")
		}
		tabStr, _ := strconv.Unquote("\"" + tabSubmatches[1] + "\"")
		var tab central.Table
		if err := json.Unmarshal([]byte(tabStr), &tab); err != nil {
			fmt.Println("TABSTR:", tabStr)
			fmt.Println("ERROR:", err)
			return 2, errors.New("unable to parse Table returned from central")
		}

		fmt.Println("TAB:", tab)

		// Determine the closest edge cluster to resolve to.
		closest, err := oe.computeClosestCluster(&tab, state)

		// TODO: Remove Extra entry before returning back home.

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

		w.WriteMsg(ret)

		return 0, nil
	}

	if upstreamErr != nil {
		return dns.RcodeServerFailure, upstreamErr
	}

	return dns.RcodeServerFailure, errNoHealthy
}

// Computes the proximity of the edge clusters running the requested service
// and returns the IP address of the closest cluster (geographically).
func (oe *OptikonEdge) computeClosestCluster(tab *central.Table, state request.Request) (string, error) {
	return "172.16.7.102", nil
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
