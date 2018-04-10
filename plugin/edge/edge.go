package edge

import (
	"net"
	"strconv"

	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

// Coords is a 2-tuple of longitude and latitude values.
type Coords [2]float64

// OptikonEdge is a plugin that returns your IP address, port and the
// protocol used for connecting to CoreDNS.
type OptikonEdge struct {
	coords    Coords
	centralIP string
	services  []string
}

// New returns a new OptikonEdge.
func New() *OptikonEdge {
	oe := &OptikonEdge{}
	return oe
}

// ServeDNS implements the plugin.Handler interface.
func (oe *OptikonEdge) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	// If the message is a response from Central, parse that response
	// differently than a user query.
	if r.Response {
		return oe.parseCentralReply(&ctx, &w, r)
	}

	// Otherwise, the requester must be a user/client.
	return oe.parseUserRequest(&ctx, &w, r)

}

// Parses a user's request to access a particular Kubernetes service.
func (oe *OptikonEdge) parseUserRequest(ctx *context.Context, w *dns.ResponseWriter, r *dns.Msg) (int, error) {

	// MAKE RECURSIVE CALL BACK TO CENTRAL IP.

	state := request.Request{W: *w, Req: r}

	a := new(dns.Msg)
	a.SetReply(state.Req)
	a.Compress = true
	a.Authoritative = true

	ip := state.IP()
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

	srv := new(dns.SRV)
	srv.Hdr = dns.RR_Header{Name: "_" + state.Proto() + "." + state.QName(), Rrtype: dns.TypeSRV, Class: state.QClass()}
	if state.QName() == "." {
		srv.Hdr.Name = "_" + state.Proto() + state.QName()
	}
	port, _ := strconv.Atoi(state.Port())
	srv.Port = uint16(port)
	srv.Target = "."

	a.Extra = []dns.RR{rr, srv}

	state.SizeAndDo(a)
	state.W.WriteMsg(a)

	return 0, nil

}

// Parses the DNS reply back from the central cluster.
func (oe *OptikonEdge) parseCentralReply(ctx *context.Context, w *dns.ResponseWriter, r *dns.Msg) (int, error) {

	state := request.Request{W: *w, Req: r}

	a := new(dns.Msg)
	a.SetReply(state.Req)
	a.Compress = true
	a.Authoritative = true

	ip := state.IP()
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

	srv := new(dns.SRV)
	srv.Hdr = dns.RR_Header{Name: "_" + state.Proto() + "." + state.QName(), Rrtype: dns.TypeSRV, Class: state.QClass()}
	if state.QName() == "." {
		srv.Hdr.Name = "_" + state.Proto() + state.QName()
	}
	port, _ := strconv.Atoi(state.Port())
	srv.Port = uint16(port)
	srv.Target = "."

	a.Extra = []dns.RR{rr, srv}

	state.SizeAndDo(a)
	state.W.WriteMsg(a)

	return 0, nil

}

// Name implements the Handler interface.
func (oe *OptikonEdge) Name() string { return "optikon-edge" }

// SetCentralIP sets the IP address for the central cluster.
func (oe *OptikonEdge) SetCentralIP(ip string) {
	oe.centralIP = ip
}

// SetLon sets the edge site longitude.
func (oe *OptikonEdge) SetLon(v float64) {
	oe.coords[0] = v
}

// SetLat sets the edge site latitude.
func (oe *OptikonEdge) SetLat(v float64) {
	oe.coords[1] = v
}
