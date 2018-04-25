package edge

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/coredns/coredns/plugin/pkg/up"

	"github.com/miekg/dns"
)

const (
	tcpTLS              = "tcp-tls"
	dialTimeout         = 4 * time.Second
	timeout             = 2 * time.Second
	healthCheckDuration = 500 * time.Millisecond
	pushPort            = "8053"
	pushProtocol        = "http"
)

// Proxy defines an upstream host.
type Proxy struct {
	addr   string
	client *dns.Client

	// Connection caching.
	expire    time.Duration
	transport *transport

	// Health checking.
	probe *up.Probe
	fails uint32

	// Service push connection.
	pushAddr string
	pushChan chan struct{}
}

// NewProxy returns a new proxy.
func NewProxy(addr string, tlsConfig *tls.Config) *Proxy {
	u, err := url.Parse(addr)
	if err != nil {
		log.Fatalf("could not parse upstream network address (%v)", err)
	}
	host, _, _ := net.SplitHostPort(u.Host)
	p := &Proxy{
		addr:      addr,
		fails:     0,
		probe:     up.New(),
		transport: newTransport(addr, tlsConfig),
		pushAddr:  newPushAddr(host),
	}
	p.client = dnsClient(tlsConfig)
	return p
}

// dnsClient returns a client used for health checking.
func dnsClient(tlsConfig *tls.Config) *dns.Client {
	c := new(dns.Client)
	c.Net = "udp"
	// TODO(miek): this should be half of healthCheckDuration?
	c.ReadTimeout = 1 * time.Second
	c.WriteTimeout = 1 * time.Second
	if tlsConfig != nil {
		c.Net = tcpTLS
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

// Down returns true if this proxy is down, i.e. has *more* fails than maxUpstreamFails.
func (p *Proxy) Down(maxUpstreamFails uint32) bool {
	if maxUpstreamFails == 0 {
		return false
	}
	fails := atomic.LoadUint32(&p.fails)
	return fails > maxUpstreamFails
}

// Stops the health checking and service pushing goroutines.
func (p *Proxy) close() {
	close(p.pushChan)
	p.probe.Stop()
	p.transport.Stop()
}

// Starts the proxy's healthchecking.
func (p *Proxy) start(healthCheckDuration time.Duration) {
	p.probe.Start(healthCheckDuration)
}

// Creates the network address for pushing service updates.
func newPushAddr(host string) string {
	return fmt.Sprintf("%s://%s:%s", pushProtocol, host, pushPort)
}

// Starts the process of pushing the list of services to upstream proxies.
func (p *Proxy) startPushingServices(servicePushDuration time.Duration, meta EdgeSite, update *ConcurrentSet) {
	ticker := time.NewTicker(servicePushDuration)
	p.pushChan = make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				if update.Len() == 0 {
					log.Infoln("no running services to push upstream")
					continue
				}
				jsn, err := convertToServiceTableUpdate(update, meta)
				if err != nil {
					log.Errorf("error while marshalling json (%v)\n", err)
					continue
				}
				req, err := http.NewRequest("POST", p.pushAddr, bytes.NewBuffer(jsn))
				if err != nil {
					log.Errorf("error while formulating request to upstream proxy (%v)\n", err)
					continue
				}
				req.Header.Set("Content-Type", "application/json")
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					log.Errorf("error while POSTing request to upstream (%v)\n", err)
					continue
				}
				if resp.StatusCode != 200 {
					resp.Body.Close()
					log.Errorf("received a not-OK response from upstream: %d", resp.StatusCode)
					continue
				}
				resp.Body.Close()
			case <-p.pushChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// Converts the current state of the set into a JSON ServiceTableUpdate.
func convertToServiceTableUpdate(services *ConcurrentSet, meta EdgeSite) ([]byte, error) {
	services.Lock()
	defer services.Unlock()
	update := ServiceTableUpdate{
		Meta:     meta,
		Services: services.items,
	}
	return json.Marshal(update)
}
