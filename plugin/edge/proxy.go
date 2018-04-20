// NOTE: This file adopted from the existing `forward` plugin for CoreDNS.

package edge

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/coredns/coredns/plugin/pkg/up"
	"wwwin-github.cisco.com/edge/optikon-dns/plugin/central"

	"github.com/miekg/dns"
)

// Proxy defines an upstream host.
type Proxy struct {
	addr   string
	client *dns.Client

	// Connection caching
	expire    time.Duration
	transport *transport

	// health checking
	probe *up.Probe
	fails uint32

	// Daemon connection.
	pushAddr    string
	pushStopper chan struct{}
}

// NewProxy returns a new proxy.
func NewProxy(addr string, tlsConfig *tls.Config) *Proxy {
	var pAddr string
	ipRegexSubmatches := ipRegex.FindStringSubmatch(addr)
	if len(ipRegexSubmatches) >= 2 {
		pAddr = "http://" + ipRegexSubmatches[1] + ":9090"
	}
	p := &Proxy{
		addr:      addr,
		fails:     0,
		probe:     up.New(),
		transport: newTransport(addr, tlsConfig),
		pushAddr:  pAddr,
	}
	p.client = dnsClient(tlsConfig)
	return p
}

// dnsClient returns a client used for health checking.
func dnsClient(tlsConfig *tls.Config) *dns.Client {
	c := new(dns.Client)
	c.Net = "udp"
	// TODO(miek): this should be half of hcDuration?
	c.ReadTimeout = 1 * time.Second
	c.WriteTimeout = 1 * time.Second

	if tlsConfig != nil {
		c.Net = "tcp-tls"
		c.TLSConfig = tlsConfig
	}
	return c
}

// SetTLSConfig sets the TLS config in the lower p.transport.
func (p *Proxy) SetTLSConfig(cfg *tls.Config) { p.transport.SetTLSConfig(cfg) }

// SetExpire sets the expire duration in the lower p.transport.
func (p *Proxy) SetExpire(expire time.Duration) { p.transport.SetExpire(expire) }

// Dial connects to the host in p with the configured transport.
func (p *Proxy) Dial(proto string) (*dns.Conn, error) { return p.transport.Dial(proto) }

// Yield returns the connection to the pool.
func (p *Proxy) Yield(c *dns.Conn) { p.transport.Yield(c) }

// Healthcheck kicks off a round of health checks for this proxy.
func (p *Proxy) Healthcheck() { p.probe.Do(p.Check) }

// Down returns true if this proxy is down, i.e. has *more* fails than maxfails.
func (p *Proxy) Down(maxfails uint32) bool {
	if maxfails == 0 {
		return false
	}

	fails := atomic.LoadUint32(&p.fails)
	return fails > maxfails
}

// close stops the health checking goroutine.
func (p *Proxy) close() {
	close(p.pushStopper)
	p.probe.Stop()
	p.transport.Stop()
}

// start starts the proxy's healthchecking.
func (p *Proxy) start(healthCheckDuration time.Duration) {
	p.probe.Start(healthCheckDuration)
}

// Starts the process of pushing the list of services to central proxies.
func (p *Proxy) startPushingServices(servicePushDuration time.Duration, meta central.EdgeSite, update *ConcurrentStringSet) {
	ticker := time.NewTicker(servicePushDuration)
	p.pushStopper = make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				if update.Size() == 0 {
					fmt.Println("INFO no running services to push to central")
					continue
				}
				jsn, err := update.ToJSON(meta)
				if err != nil {
					fmt.Println("ERROR while marshalling JSON:", err)
					continue
				}
				req, err := http.NewRequest("POST", p.pushAddr, bytes.NewBuffer(jsn))
				if err != nil {
					fmt.Println("ERROR while formulating request to central:", err)
					continue
				}
				req.Header.Set("Content-Type", "application/json")
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					fmt.Println("ERROR while POSTing request to central:", err)
					continue
				}
				if resp.StatusCode != 200 {
					resp.Body.Close()
					fmt.Println("ERROR: Received non-200 response from central:", resp.StatusCode)
					continue
				}
				resp.Body.Close()
			case <-p.pushStopper:
				ticker.Stop()
				return
			}
		}
	}()
}

const (
	dialTimeout = 4 * time.Second
	timeout     = 2 * time.Second
	hcDuration  = 500 * time.Millisecond
)

var (
	ipRegex = regexp.MustCompile(`^(.*):\d+$`)
)
