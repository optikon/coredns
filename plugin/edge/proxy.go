package edge

import (
	"bytes"
	"crypto/tls"
	"errors"
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
	pushAddr  string
	pushProbe *up.Probe
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
		pushProbe: up.New(),
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
	p.pushProbe.Stop()
	p.probe.Stop()
	p.transport.Stop()
}

// start starts the proxy's healthchecking.
func (p *Proxy) start(healthCheckDuration, servicePushDuration time.Duration) {
	p.probe.Start(healthCheckDuration)
	p.pushProbe.Start(servicePushDuration)
}

// Starts the process of pushing the list of services to central proxies.
func (p *Proxy) startPushingServices(meta central.EdgeSite, update *ConcurrentStringSlice) {

	// Packages services into JSON and posts to central proxy.
	p.pushProbe.Do(func() error {
		jsn, err := update.ToJSON(meta)
		if err != nil {
			return err
		}
		req, err := http.NewRequest("POST", p.pushAddr, bytes.NewBuffer(jsn))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return errors.New("non-200 response from central")
		}
		return nil
	})
}

const (
	dialTimeout = 4 * time.Second
	timeout     = 2 * time.Second
	hcDuration  = 500 * time.Millisecond
)

var (
	ipRegex = regexp.MustCompile(`^(.*):\d+$`)
)
