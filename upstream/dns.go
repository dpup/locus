package upstream

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Register an upstream factor that matches hostnames on the form foo.bar.baz
func init() {
	Register(
		`^[[:alnum:]][[:alnum:]\.\-]+[[:alnum:]]$`,
		func(host string, settings map[string]string) (Source, error) {
			var port uint16 = 80
			var ttl time.Duration
			var allowStale bool

			if p, ok := settings["port"]; ok {
				pi, err := strconv.ParseInt(p, 10, 16)
				if err != nil {
					return nil, fmt.Errorf("invalid port '%s', %s", p, err)
				}
				port = uint16(pi)
			}

			if a, ok := settings["allow_stale"]; ok {
				ai, err := strconv.ParseBool(a)
				if err != nil {
					return nil, fmt.Errorf("invalid boolean for allow_stale '%s', %s", a, err)
				}
				allowStale = ai
			}

			if t, ok := settings["ttl"]; ok {
				ti, err := time.ParseDuration(t)
				if err != nil {
					return nil, fmt.Errorf("invalid duration for ttl, '%s', %s", t, err)
				}
				ttl = ti
			}

			return &DNS{
				Host:       host,
				Port:       port,
				Path:       settings["path"],
				AllowStale: allowStale,
				TTL:        ttl,
			}, nil
		})
}

// DefaultDNSTTL is 1 minute.
const DefaultDNSTTL = time.Minute

// FakeHost is hardcoded not to hit the actual DNS resolver, instead
// returning a set of local IPs.
const FakeHost = "dns.test.fake"

// DNS is an upstream source that looks up hosts from DNS.
//
// If AllowStale is true, an old list of upstreams will be used following a
// failed refresh. If AllowStale is false, the error will be propagated to
// callers.
//
// The default TTL for entries is 1 minute, to override, set the TTL field. Once
// TTL has expired, requests will block on refreshing the upstreams.
type DNS struct {
	Host       string
	Port       uint16
	Path       string
	AllowStale bool
	TTL        time.Duration

	addrs     []*url.URL
	expiresAt time.Time
	err       error
	mu        sync.Mutex
}

// DebugInfo returns extra fields to show on /debug/configs
func (ds *DNS) DebugInfo() map[string]string {
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
func (ds *DNS) All() ([]*url.URL, error) {
	ds.maybeRefresh()
	return ds.addrs, ds.err
}

func (ds *DNS) ttl() time.Duration {
	if ds.TTL == 0 {
		return DefaultDNSTTL
	}
	return ds.TTL
}

func (ds *DNS) maybeRefresh() {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	now := time.Now()
	if len(ds.addrs) != 0 && now.Before(ds.expiresAt) {
		return
	}

	var addrs []string
	if ds.Host == FakeHost {
		addrs = []string{"192.168.0.0", "192.168.0.1", "192.168.0.2", "192.168.0.3"}
	} else {
		var err error
		addrs, err = net.LookupHost(ds.Host)
		if err != nil {
			if ds.AllowStale && len(ds.addrs) != 0 {
				log.Printf("error looking up %s, using stale upstreams", ds.Host)
			} else {
				ds.addrs = nil
				ds.err = err
			}
			return
		}
	}

	log.Printf("dns refreshed for %s, %d upstream(s) found", ds.Host, len(addrs))

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
			Path:   ds.Path,
		}
	}

	ds.err = nil
	ds.expiresAt = now.Add(ds.ttl())
}
