package edge

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
)

// Edge encapsulates all edge plugin state.
type Edge struct {

	// Next is a reference to the next plugin in the CoreDNS plugin chain.
	Next plugin.Handler

	// Table stores the service->[]edgesite mappings for this and all
	// downstream edge sites.
	table *ConcurrentTable

	// Clientset is a reference to in-cluster Kubernetes API.
	clientset *kubernetes.Clientset

	// IP is the public IP address of this cluster.
	ip string

	// The geo coordinates of this cluster.
	geoCoords *Point

	// The interval for reading and pushing locally running Kubernetes services.
	svcReadInterval time.Duration
	svcPushInterval time.Duration

	// A channel for halting the service-reading process.
	svcReadChan chan struct{}

	// A server for receiving table updates from downstream edge sites.
	server *http.Server

	// The set of services currently running at this edge site.
	services *ConcurrentStringSet

	// TODO: Clean.
	proxies       []*Proxy
	p             Policy
	hcInterval    time.Duration
	from          string
	ignored       []string
	tlsConfig     *tls.Config
	tlsServerName string
	maxfails      uint32
	expire        time.Duration
	forceTCP      bool
}

// New returns a new Edge instance.
func New() *Edge {
	return &Edge{
		// TODO: CLEAN
		maxfails:   2,
		tlsConfig:  new(tls.Config),
		expire:     defaultExpire,
		p:          new(random),
		from:       ".",
		hcInterval: hcDuration,
		table:      NewConcurrentTable(),
		services:   NewConcurrentStringSet(),
	}
}

// Len returns the number of configured proxies.
// TODO: Clean.
func (e *Edge) Len() int { return len(e.proxies) }

// Name implements the plugin.Handler interface.
func (e *Edge) Name() string { return pluginName }

// ServeDNS implements the plugin.Handler interface.
func (e *Edge) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	// Encapsolate the state of the request and response.
	state := request.Request{W: w, Req: r}

	// Declare the response we want to send back.
	res := new(dns.Msg)

	// Parse out the LOC field from the request, if one exists.
	loc, locFound := parseLoc(r.Extra)

	// Parse the target domain out of the request (NOTE: This will always have
	// a trailing dot.)
	requestedService := state.Name().TrimTrailingDot()

	//
	// TODO: Determine if the request has an Extra LOC field. If it does, then
	// return the IP of the site running the service closest to LOC geo coords.
	//
	// IF there's no LOC field, this is a client request, in which case, check
	// if the service is running locally. If it is, return my ip. If it isn't,
	// Determine the IP of the site running the service closest to ME, using my
	// local table.
	//
	// If the service can't be found in my table, forward the request to all
	// my proxies, and send my LOC as the Extra field. Whatever response I get
	// back, return it to the client, and hopefully the cache plugin will cache
	// the reply.
	//

	// Determine if the requested service is running locally and write a reply
	// with my ip if it is.
	if !locFound && e.services.Contains(requestedService) {
		writeAuthoritativeResponse(res, &state, o.ip)
		return dns.RcodeSuccess, nil
	}

	// Determine if there is another edge site that I know of that is running
	// the requested service. If there is, redirect to the closest.
	edgeSites, entryFound := e.table.Lookup(requestedService)
	if entryFound && len(edgeSites) > 0 {
		var closest string
		if locFound {
			closest = edgeSites.FindClosestToPoint(loc)
		} else {
			closest = edgeSites.FindClosestToPoint(o.geoCoords)
		}
		writeAuthoritativeResponse(res, &state, closest)
		return dns.RcodeSuccess, nil
	}

	//
	// FORWARD REQUEST.
	// TODO: CLEAN
	//
	// TODO: MAKE SURE THIS IS OKAY WITH HAVING NO UPSTREAMS.
	// If there are no upstreams, fall through to proxy plugin.
	// If there are upstreams, forward to them.
	//

	if !e.match(state) {
		// TODO: Is this right?
		return plugin.NextOrFailure(e.Name(), e.Next, ctx, w, r)
	}

	fails := 0
	var span, child ot.Span
	var upstreamErr error
	span = ot.SpanFromContext(ctx)

	for _, proxy := range e.list() {
		if proxy.Down(e.maxfails) {
			fails++
			if fails < len(e.proxies) {
				continue
			}
			// All upstream proxies are dead, assume healtcheck is completely broken and randomly
			// select an upstream to connect to.
			r := new(random)
			proxy = r.List(e.proxies)[0]
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
			ret, err = proxy.connect(ctx, state, e.forceTCP, true)
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
			if e.maxfails != 0 {
				proxy.Healthcheck()
			}

			if fails < len(e.proxies) {
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
			if len(ret.Answer) == 0 {
				return dns.RcodeServerFailure, errTableParseFailure
			}
			w.WriteMsg(ret)
			return 0, nil
		}

		// Extract the edge sites from the response.
		edgeSiteRR := ret.Extra[0]
		edgeSiteSubmatches := edgeSiteRegex.FindStringSubmatch(edgeSiteRR.String())
		if len(edgeSiteSubmatches) < 2 {
			return dns.RcodeServerFailure, errTableParseFailure
		}
		edgeSiteStr, err := strconv.Unquote(fmt.Sprintf("\"%s\"", edgeSiteSubmatches[1]))
		if err != nil {
			return dns.RcodeServerFailure, errTableParseFailure
		}
		var edgeSites []EdgeSite
		if err := json.Unmarshal([]byte(edgeSiteStr), &edgeSites); err != nil {
			return dns.RcodeServerFailure, errTableParseFailure
		}

		// Remove the Table entry from the return message.
		ret.Extra = ret.Extra[1:]

		// If the list is empty, call the next plugin (proxy).
		if len(edgeSites) == 0 {
			return plugin.NextOrFailure(e.Name(), e.Next, ctx, w, r)
		}

		// Compute the distance to the first edge site.
		closest := edgeSites[0].IP
		minDist := Distance(e.lat, e.lon, edgeSites[0].Lat, edgeSites[0].Lon)
		for _, edgeSite := range edgeSites {
			dist := Distance(e.lat, e.lon, edgeSite.Lat, edgeSite.Lon)
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

// Write the given IP address as an Authoritative Answer to the request.
func writeAuthoritativeResponse(res *dns.Msg, state *request.Request, string ip) {

	// Set the reply to the given request.
	res.SetReply(state.Req)

	// Make the answer Authoritative and compressed.
	res.Authoritative, res.Compress = true, true

	// Add the IP address to the Answer field.
	var rr dns.RR
	switch state.Family() {
	case 1:
		rr = new(dns.A)
		rr.(*dns.A).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass()}
		rr.(*dns.A).A = net.ParseIP(ip).To4()
	case 2:
		rr = new(dns.AAAA)
		rr.(*dns.AAAA).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeAAAA, Class: state.QClass()}
		rr.(*dns.AAAA).AAAA = net.ParseIP(ip)
	}
	res.Answer = []dns.RR{rr}

	// Write the message.
	state.W.WriteMsg(res)
}

func (e *Edge) checkRunningLocally(w dns.ResponseWriter, r *dns.Msg) (int, error, bool) {

	// Encapsolate the state of the request and response.
	state := request.Request{W: w, Req: r}

	// Parse the target domain out of the request (NOTE: This will always have
	// a trailing dot.)
	targetDomain := state.Name()

	// If we're already running the service, return my IP.
	if e.services.Contains(targetDomain[:(len(targetDomain) - 1)]) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Authoritative, ret.RecursionAvailable, ret.Compress = true, true, true
		var rr dns.RR
		switch state.Family() {
		case 1:
			rr = new(dns.A)
			rr.(*dns.A).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass()}
			rr.(*dns.A).A = net.ParseIP(e.ip).To4()
		case 2:
			rr = new(dns.AAAA)
			rr.(*dns.AAAA).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeAAAA, Class: state.QClass()}
			rr.(*dns.AAAA).AAAA = net.ParseIP(e.ip)
		}
		ret.Answer = []dns.RR{rr}
		w.WriteMsg(ret)
		return 0, nil
	}

}

func (e *Edge) match(state request.Request) bool {
	from := e.from

	if !plugin.Name(from).Matches(state.Name()) || !e.isAllowedDomain(state.Name()) {
		return false
	}

	return true
}

func (e *Edge) isAllowedDomain(name string) bool {
	if dns.Name(name) == dns.Name(e.from) {
		return true
	}

	for _, ignore := range e.ignored {
		if plugin.Name(ignore).Matches(name) {
			return false
		}
	}
	return true
}

// List returns a set of proxies to be used for this client depending on the policy in e.
func (e *Edge) list() []*Proxy { return e.p.List(e.proxies) }

var (
	errInvalidDomain         = errors.New("invalid domain for forward")
	errNoHealthy             = errors.New("no healthy proxies")
	errNoEdge                = errors.New(fmt.Springf("no %s defined", pluginName))
	errTableParseFailure     = errors.New("unable to parse Table returned from upstream")
	errFindingClosestCluster = errors.New("unable to compute closest edge cluster")
)

// policy tells forward what policy for selecting upstream it uses.
type policy int

const (
	randomPolicy policy = iota
	roundRobinPolicy
)

var (
	edgeSiteRegex = regexp.MustCompile(`^.*\t0\tIN\tTXT\t\"(\[.*\])\"$`)
)

// The name of the plugin, as seen by CoreDNS.
const pluginName = "edge"
