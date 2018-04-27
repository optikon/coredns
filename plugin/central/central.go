package central

import (
	"encoding/json"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

// Table specifies the mapping from service DNS names to edge sites.
type Table map[string][]EdgeSite

// EdgeSite is a wrapper around all information needed about edge sites serving
// content.
type EdgeSite struct {
	IP  string  `json:"ip"`
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

// OptikonCentral is a plugin that returns your IP address, port and the
// protocol used for connecting to CoreDNS.
type OptikonCentral struct {
	table Table
	Next  plugin.Handler
}

// New returns a new OptikonCentral.
func New() *OptikonCentral {
	oc := &OptikonCentral{
		table: make(Table),
	}
	return oc
}

func (oc *OptikonCentral) populateTable() {

	oc.table["kubernetes.default.svc.cluster.external"] = []EdgeSite{
		EdgeSite{
			IP:  "172.16.7.102",
			Lon: 55.664023,
			Lat: 12.610126,
		},
		EdgeSite{
			IP:  "172.16.7.103",
			Lon: 55.680770,
			Lat: 12.543006,
		},
		EdgeSite{
			IP:  "172.16.7.104",
			Lon: 55.6748923,
			Lat: 12.5534,
		},
	}

	oc.table["nginx-kubecon.default.svc.cluster.external"] = []EdgeSite{
		EdgeSite{
			IP:  "172.16.7.102",
			Lon: 55.664023,
			Lat: 12.610126,
		},
		EdgeSite{
			IP:  "172.16.7.103",
			Lon: 55.680770,
			Lat: 12.543006,
		},
	}
}

// ServeDNS implements the plugin.Handler interface.
func (oc *OptikonCentral) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	// Encapsolate the state of the request and reponse.
	state := request.Request{W: w, Req: r}

	// Parse the target domain out of the request (NOTE: This will always have
	// a trailing dot.)
	targetDomain := state.Name()

	// Determine if there is an entry for the DNS name we're looking for.
	edgeSites, found := oc.table[targetDomain[:(len(targetDomain)-1)]]
	if !found || len(edgeSites) == 0 {
		return plugin.NextOrFailure(oc.Name(), oc.Next, ctx, w, r)
	}

	// Convert the edge sites to a JSON string.
	jsonString, err := json.Marshal(edgeSites)
	if err != nil {
		return dns.RcodeServerFailure, err
	}

	// Init a response message.
	res := new(dns.Msg)
	res.SetReply(r)
	res.Compress = true
	res.Authoritative = false
	res.Response = true

	// Initialze a text resource record (RR) for the edge sites.
	es := new(dns.TXT)
	es.Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeTXT, Class: state.QClass()}
	es.Txt = []string{string(jsonString)}

	// Send it as part of the Extra/Additional field of the DNS packet.
	res.Extra = []dns.RR{es}

	// Write the response message.
	state.SizeAndDo(res)
	w.WriteMsg(res)

	// Return no errors.
	return dns.RcodeSuccess, nil
}

// Name implements the Handler interface.
func (oc *OptikonCentral) Name() string { return "optikon-central" }
