package edge

import (
	"sync/atomic"

	"github.com/miekg/dns"
)

// For HC we send to . IN NS +norec message to the upstream. Dial timeouts and empty
// replies are considered fails, basically anything else constitutes a healthy upstream.

// Check is used as the up.Func in the up.Probe.
func (p *Proxy) Check() error {
	err := p.sendHealthCheck()
	if err != nil {
		atomic.AddUint32(&p.fails, 1)
		return err
	}
	atomic.StoreUint32(&p.fails, 0)
	return nil
}

// Sends a healthcheck ping to the proxy.
func (p *Proxy) sendHealthCheck() error {
	hcping := new(dns.Msg)
	hcping.SetQuestion(".", dns.TypeNS)
	m, _, err := p.client.Exchange(hcping, p.addr)
	if err != nil && m != nil {
		if m.Response || m.Opcode == dns.OpcodeQuery {
			err = nil
		}
	}
	return err
}
