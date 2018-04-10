package central

import (
	"net"
	"strconv"

	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

// EdgeSite is a wrapper around all information needed about edge sites serving
// content.
type EdgeSite struct {
	ip  string
	lon float64
	lat float64
}

// Table specifies the mapping from service DNS names to edge sites.
type Table map[string][]*EdgeSite

// OptikonCentral is a plugin that returns your IP address, port and the
// protocol used for connecting to CoreDNS.
type OptikonCentral struct {
	table Table
}

// New returns a new OptikonCentral.
func New() *OptikonCentral {
	oc := &OptikonCentral{
		table: make(Table),
	}
	return oc
}

func (oc *OptikonCentral) populateTable() {
	oc.table["echoserver"] = []*EdgeSite{
		&EdgeSite{
			ip:  "172.16.7.102",
			lon: 55.680770,
			lat: 12.543006,
		},
	}
}

// ServeDNS implements the plugin.Handler interface.
func (oc *OptikonCentral) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	state := request.Request{W: w, Req: r}

	a := new(dns.Msg)
	a.SetReply(r)
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
	w.WriteMsg(a)

	return 0, nil
}

// Name implements the Handler interface.
func (oc *OptikonCentral) Name() string { return "optikon-central" }
