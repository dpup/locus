package locus

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// UpstreamProvider defines an interface for fetching sets of upstream servers,
// and selecting a candidate for the request.
type UpstreamProvider interface {
	// Get returns a single URL that can be used to make a request to.
	Get(req *http.Request) (*url.URL, error)

	// All returns all the known upstream URLs.
	All() ([]*url.URL, error)
}

// Single returns a provider that only has one upstream.
func Single(urlStr string) UpstreamProvider {
	return &fixedSet{urlStrs: []string{urlStr}, pickFn: first}
}

// Random returns an upstream provider which picks a random URL from a fixed
// set of URLs. Example use:
//     cfg.Upstream(Random([]string{
//       "back-1.test.com",
//       "back-2.test.com",
//       "back-3.test.com",
//     }))
func Random(urlStrs []string) UpstreamProvider {
	return &fixedSet{urlStrs: urlStrs, pickFn: random}
}

// RoundRobin returns an upstream provider which cycles through the set of URLs.
// Example use:
//     cfg.Upstream(RoundRobin([]string{
//       "back-1.test.com",
//       "back-2.test.com",
//       "back-3.test.com",
//     }))
func RoundRobin(urlStrs []string) UpstreamProvider {
	return &fixedSet{urlStrs: urlStrs, pickFn: roundRobin(len(urlStrs))}
}

// fixedSet implements UpstreamProvider, storing a fixed set of URLs, and
// caches parsed URLs and any error that occurred.
type fixedSet struct {
	urlStrs []string
	pickFn  func(urls []*url.URL) *url.URL

	urls []*url.URL
	err  error
	mu   sync.Mutex
}

// Get returns a random upstream.
func (ru *fixedSet) Get(req *http.Request) (*url.URL, error) {
	if ru.urls == nil && ru.err == nil {
		ru.parseURLs()
	}
	if ru.err != nil {
		return nil, ru.err
	}
	return ru.pickFn(ru.urls), nil
}

// All returns all upsteams.
func (ru *fixedSet) All() ([]*url.URL, error) {
	if ru.urls == nil && ru.err == nil {
		ru.parseURLs()
	}
	return ru.urls, ru.err
}

func (ru *fixedSet) parseURLs() {
	ru.mu.Lock()
	defer ru.mu.Unlock()
	if ru.urls != nil || ru.err != nil {
		// Calculation done in a racing thread.
		return
	}
	urls := make([]*url.URL, len(ru.urlStrs))
	for i, urlStr := range ru.urlStrs {
		u, err := url.Parse(string(urlStr))
		if err != nil {
			ru.err = fmt.Errorf("unable to parse '%s': %s", urlStr, err)
			return
		}
		urls[i] = u
	}
	ru.urls = urls
}

// DNS returns an upstream provider that looksup upstream hosts via DNS.
//
// Example use:
//     cfg.Upstream(DNS("amazon.com", 80, "/"))
//
func DNS(dnsHost string, port uint16, pathPrefix string) UpstreamProvider {
	return &DNSSet{DNSHost: dnsHost, Port: port, PathPrefix: pathPrefix}
}

// DefaultDNSTTL is 1 minute.
const DefaultDNSTTL = time.Minute

// FakeDNSHost is hardcoded not to hit the actual DNS resolver, instead
// returning a set of local IPs.
const FakeDNSHost = "dns.test.fake"

// DNSSet exposes the UpstreamProvider interface, fetching upstream hosts from
// DNS.
//
// If RoundRobin is true, upstream URLs will be used in order. Otherwise a
// random server will be picked for each request.
//
// If AllowStale is true, an old list of upstreams will be used following a
// failed refresh. If AllowStale is false, the error will be propagated to
// callers.
//
// The default TTL for entries is 1 minute, to override set the TTL field. Once
// TTL has expired, requests will block on refreshing the upstreams.
type DNSSet struct {
	DNSHost    string
	Port       uint16
	PathPrefix string
	RoundRobin bool
	AllowStale bool
	TTL        time.Duration

	pickFn    func(urls []*url.URL) *url.URL
	addrs     []*url.URL
	expiresAt time.Time
	err       error
	mu        sync.Mutex
}

// Get returns a random upstream.
func (ds *DNSSet) Get(req *http.Request) (*url.URL, error) {
	ds.maybeRefresh()
	if ds.err != nil {
		return nil, ds.err
	}
	return ds.pickFn(ds.addrs), nil
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

	if ds.RoundRobin {
		ds.pickFn = roundRobin(len(ds.addrs))
	} else {
		ds.pickFn = random
	}
}

// first returns the first entry in the array.
func first(urls []*url.URL) *url.URL {
	return urls[0]
}

// random pickes a random entry from the array.
func random(urls []*url.URL) *url.URL {
	return urls[rand.Intn(len(urls))]
}

// roundRobin returns a method that uses round robin to pick the next entry.
func roundRobin(count int) func(urls []*url.URL) *url.URL {
	ch := make(chan int, 1)
	go func() {
		c := 0
		for {
			ch <- c
			c++
			if c == count {
				c = 0
			}
		}
	}()
	return func(urls []*url.URL) *url.URL {
		return urls[<-ch]
	}
}
