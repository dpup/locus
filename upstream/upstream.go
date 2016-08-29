// Package upstream includes several implementations of locus.UpstreamProvider.
package upstream

import (
	"hash/fnv"
	"math/rand"
	"net/http"
	"net/url"
)

// PickFn is used to select a URL from an array of URLs.
type PickFn func(urls []*url.URL) *url.URL

// Provider defines an interface for fetching an upstream to forward a
// request too. Providers must also satisfy the Source interface.
type Provider interface {
	Source

	// Get returns a single URL that can be used to make a request to.
	Get(req *http.Request) (*url.URL, error)
}

// Source defines an interface for fetching sets of upstream servers.
type Source interface {
	// All returns all the known upstream URLs.
	All() ([]*url.URL, error)

	// DebugInfo returns information about the upstream, for presentation on debug
	// screens.
	DebugInfo() map[string]string
}

// Single returns a provider that only has one upstream.
func Single(urlStr string) Provider {
	return First(FixedSet(urlStr))
}

// First returns an upstream Provider that always uses the first upstream in a
// Source.
func First(source Source) Provider {
	return &provider{Source: source, pickFn: func(urls []*url.URL) *url.URL {
		return urls[0]
	}}
}

// Random returns an Provider that picks a random upstream from a Source.
func Random(source Source) Provider {
	return &provider{Source: source, pickFn: func(urls []*url.URL) *url.URL {
		return urls[rand.Intn(len(urls))]
	}}
}

// RoundRobin returns an Provider that cycles through the upstreams in a Source.
func RoundRobin(source Source) Provider {
	ch := make(chan int, 1)
	go func() {
		c := 0
		for {
			ch <- c
			c++
		}
	}()
	return &provider{Source: source, pickFn: func(urls []*url.URL) *url.URL {
		return urls[<-ch%len(urls)]
	}}
}

// provider composes a Source, satisfying the upstream Provider interface.
type provider struct {
	Source
	pickFn PickFn
}

func (p *provider) Get(req *http.Request) (*url.URL, error) {
	urls, err := p.All()
	if err != nil {
		return nil, err
	}
	return p.pickFn(urls), nil
}

// IPHash returns an Provider that sends traffic to a consistent backend based
// on a hash of the requesting IP (via X-Forwarded-For or Remote_Addr).
func IPHash(source Source) Provider {
	return &ipHashProvider{Source: source}
}

type ipHashProvider struct {
	Source
}

func (p *ipHashProvider) Get(req *http.Request) (*url.URL, error) {
	urls, err := p.All()
	if err != nil {
		return nil, err
	}

	h := fnv.New32()
	h.Write([]byte(clientIP(req)))

	return urls[int(h.Sum32())%len(urls)], nil
}

func clientIP(req *http.Request) string {
	if h := req.Header.Get("X-Forwarded-For"); h != "" {
		return h
	}
	return req.RemoteAddr
}
