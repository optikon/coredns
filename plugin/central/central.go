package central

import (
	"encoding/json"

	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
	"wwwin-github.cisco.com/edge/optikon-dns/plugin/edge"
)

// Table specifies the mapping from service DNS names to edge sites.
type Table map[string][]edge.Site

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
	oc.table["echoserver.default.svc.cluster.local"] = []edge.Site{
		edge.Site{
			IP:  "172.16.7.102",
			Lon: 55.680770,
			Lat: 12.543006,
		},
	}
}

// ServeDNS implements the plugin.Handler interface.
func (oc *OptikonCentral) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	// Convert the Table to a JSON string.
	jsonString, err := json.Marshal(oc.table)
	if err != nil {
		return 2, err
	}

	// Encapsolate the state of the request and reponse.
	state := request.Request{W: w, Req: r}

	// Init a response message.
	res := new(dns.Msg)
	res.SetReply(r)
	res.Compress = true
	res.Authoritative = false
	res.Response = true

	// Initialze a text resource record (RR) for the Table.
	tab := new(dns.TXT)
	tab.Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeTXT, Class: state.QClass()}
	tab.Txt = []string{string(jsonString)}

	// Send it as part of the Extra/Additional field of the DNS packet.
	res.Extra = []dns.RR{tab}

	// Write the response message.
	state.SizeAndDo(res)
	w.WriteMsg(res)

	// Return no errors.
	return 0, nil
}

// Name implements the Handler interface.
func (oc *OptikonCentral) Name() string { return "optikon-central" }
