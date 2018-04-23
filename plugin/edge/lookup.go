/*
 * Copyright 2018 The CoreDNS Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may
 * not use this file except in compliance with the License. You may obtain
 * a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package edge

import (
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

// Forward forward the request in state as-is. Unlike Lookup that adds EDNS0 suffix to the message.
// Forward may be called with a nil f, an error is returned in that case.
func (oe *OptikonEdge) Forward(state request.Request) (*dns.Msg, error) {
	if oe == nil {
		return nil, errNoOptikonEdge
	}

	fails := 0
	var upstreamErr error
	for _, proxy := range oe.list() {
		if proxy.Down(oe.maxfails) {
			fails++
			if fails < len(oe.proxies) {
				continue
			}
			// All upstream proxies are dead, assume healtcheck is complete broken and randomly
			// select an upstream to connect to.
			proxy = oe.list()[0]
		}

		ret, err := proxy.connect(context.Background(), state, oe.forceTCP, true)

		ret, err = truncated(ret, err)
		upstreamErr = err

		if err != nil {
			if fails < len(oe.proxies) {
				continue
			}
			break
		}

		// Check if the reply is correct; if not return FormErr.
		if !state.Match(ret) {
			return state.ErrorMessage(dns.RcodeFormatError), nil
		}

		return ret, err
	}

	if upstreamErr != nil {
		return nil, upstreamErr
	}

	return nil, errNoHealthy
}

// Lookup will use name and type to forge a new message and will send that upstream. It will
// set any EDNS0 options correctly so that downstream will be able to process the reply.
// Lookup may be called with a nil f, an error is returned in that case.
func (oe *OptikonEdge) Lookup(state request.Request, name string, typ uint16) (*dns.Msg, error) {
	if oe == nil {
		return nil, errNoOptikonEdge
	}

	req := new(dns.Msg)
	req.SetQuestion(name, typ)
	state.SizeAndDo(req)

	state2 := request.Request{W: state.W, Req: req}

	return oe.Forward(state2)
}

// NewLookup returns a OptikonEdge that can be used for plugin that need an upstream to resolve external names.
// Note that the caller must run Close on the forward to stop the health checking goroutines.
func NewLookup(addr []string) *OptikonEdge {
	oe := New()
	for i := range addr {
		p := NewProxy(addr[i], nil)
		oe.SetProxy(p)
	}
	return oe
}
