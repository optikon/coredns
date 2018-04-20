package central

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
)

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
	table           *ConcurrentTable
	Next            plugin.Handler
	clientset       *kubernetes.Clientset
	ip              string
	lon             float64
	lat             float64
	svcReadInterval time.Duration
	svcReadStopper  chan struct{}
	server          *http.Server
}

// New returns a new OptikonCentral.
func New() *OptikonCentral {
	oc := &OptikonCentral{
		table: NewConcurrentTable(),
	}
	return oc
}

// ServeDNS implements the plugin.Handler interface.
func (oc *OptikonCentral) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	// Encapsolate the state of the request and reponse.
	state := request.Request{W: w, Req: r}

	// Parse the target domain out of the request (NOTE: This will always have
	// a trailing dot.)
	targetDomain := state.Name()

	// Determine if there is an entry for the DNS name we're looking for.
	// If not, fall through to the proxy plugin.
	edgeSites, found := oc.table.Lookup(targetDomain[:(len(targetDomain) - 1)])
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
