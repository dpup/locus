package upstream

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"sync"
	"time"
)

// DNS returns an upstream source that looks up hosts via DNS.
//
// Example use:
//     cfg.Upstream(Random(DNS("amazon.com", 80, "/")))
//
func DNS(dnsHost string, port uint16, pathPrefix string) *DNSSet {
	return &DNSSet{DNSHost: dnsHost, Port: port, PathPrefix: pathPrefix}
}

// DefaultDNSTTL is 1 minute.
const DefaultDNSTTL = time.Minute

// FakeDNSHost is hardcoded not to hit the actual DNS resolver, instead
// returning a set of local IPs.
const FakeDNSHost = "dns.test.fake"

// DNSSet is an upstream source that looks up hosts from DNS.
//
// If AllowStale is true, an old list of upstreams will be used following a
// failed refresh. If AllowStale is false, the error will be propagated to
// callers.
//
// The default TTL for entries is 1 minute, to override, set the TTL field. Once
// TTL has expired, requests will block on refreshing the upstreams.
type DNSSet struct {
	DNSHost    string
	Port       uint16
	PathPrefix string
	AllowStale bool
	TTL        time.Duration

	addrs     []*url.URL
	expiresAt time.Time
	err       error
	mu        sync.Mutex
}

// DebugInfo returns extra fields to show on /debug/configs
func (ds *DNSSet) DebugInfo() map[string]string {
	m := map[string]string{}
	if ds.err != nil {
		m["error"] = ds.err.Error()
	}
	m["allow stale"] = fmt.Sprintf("%v", ds.AllowStale)
	m["TTL"] = ds.ttl().String()
	m["expires at"] = ds.expiresAt.Format(time.Stamp)
	return m
}

// All returns all upsteams.
func (ds *DNSSet) All() ([]*url.URL, error) {
	ds.maybeRefresh()
	return ds.addrs, ds.err
}

func (ds *DNSSet) ttl() time.Duration {
	if ds.TTL == 0 {
		return DefaultDNSTTL
	}
	return ds.TTL
}

func (ds *DNSSet) maybeRefresh() {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	now := time.Now()
	if len(ds.addrs) != 0 && now.Before(ds.expiresAt) {
		return
	}

	var addrs []string
	if ds.DNSHost == FakeDNSHost {
		addrs = []string{"192.168.0.0", "192.168.0.1", "192.168.0.2", "192.168.0.3"}
	} else {
		var err error
		addrs, err = net.LookupHost(ds.DNSHost)
		if err != nil {
			if ds.AllowStale && len(ds.addrs) != 0 {
				log.Printf("error looking up %s, using stale upstreams", ds.DNSHost)
			} else {
				ds.addrs = nil
				ds.err = err
			}
			return
		}
	}

	log.Printf("dns refreshed for %s, %d upstream(s) found", ds.DNSHost, len(addrs))

	ds.addrs = make([]*url.URL, len(addrs))
	for i, addr := range addrs {
		var scheme string
		if ds.Port == 443 {
			scheme = "https"
		} else {
			scheme = "http"
		}
		ds.addrs[i] = &url.URL{
			Scheme: scheme,
			Host:   fmt.Sprintf("%s:%d", addr, ds.Port),
			Path:   ds.PathPrefix,
		}
	}

	ds.err = nil
	ds.expiresAt = now.Add(ds.ttl())
}
