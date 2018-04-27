// Adapted from https://github.com/coredns/coredns/blob/master/plugin/forward/connect.go

package edge

import (
	"time"

	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

// Establishes a connection and forwards a message to the upstream proxy.
func (p *Proxy) connect(ctx context.Context, state request.Request, forceTCP, metric bool) (*dns.Msg, error) {

	proto := state.Proto()
	if forceTCP {
		proto = "tcp"
	}

	conn, err := p.Dial(proto)
	if err != nil {
		return nil, err
	}

	// Set buffer size correctly for this client.
	conn.UDPSize = uint16(state.Size())
	if conn.UDPSize < 512 {
		conn.UDPSize = 512
	}

	conn.SetWriteDeadline(time.Now().Add(timeout))
	if err := conn.WriteMsg(state.Req); err != nil {
		conn.Close() // not giving it back
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(timeout))
	ret, err := conn.ReadMsg()
	if err != nil {
		conn.Close() // not giving it back
		return nil, err
	}

	p.Yield(conn)

	return ret, nil
}
