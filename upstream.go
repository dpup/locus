package locus

import (
	"math/rand"
	"net/http"
	"net/url"
	"sync"
)

// UpstreamProvider defines an interface for fetching sets of upstream servers,
// and selecting a candidate for the request.
type UpstreamProvider interface {
	// Get returns a single upstream, to use to make a request to.
	Get(req *http.Request) (*url.URL, error)

	// All returns all, currently known upstreams.
	All(req *http.Request) ([]*url.URL, error)
}

// Single returns a provider that only has one upstream.
// For example:
//     cfg.Upstream(Single("http://test.com"))
func Single(urlStr string) UpstreamProvider {
	return &fixedSet{urlStrs: []string{urlStr}, pick: func(urls []*url.URL) *url.URL {
		return urls[0]
	}}
}

// Random returns an upstream provider which picks a random URL from a fixed
// set of URLs.
// For example:
//     cfg.Upstream(Random([]string{
//       "back-1.test.com",
//       "back-2.test.com",
//       "back-3.test.com",
//     }))
func Random(urlStrs []string) UpstreamProvider {
	return &fixedSet{urlStrs: urlStrs, pick: func(urls []*url.URL) *url.URL {
		return urls[rand.Intn(len(urls))]
	}}
}

// RoundRobin returns an upstream provider which cycles through the set of URLs.
// For example:
//     cfg.Upstream(RoundRobin([]string{
//       "back-1.test.com",
//       "back-2.test.com",
//       "back-3.test.com",
//     }))
func RoundRobin(urlStrs []string) UpstreamProvider {
	return &fixedSet{urlStrs: urlStrs, pick: rrPicker(len(urlStrs))}
}

// rrPicker returns a method that uses round robin to pick the next entry.
func rrPicker(count int) func(urls []*url.URL) *url.URL {
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

// fixedSet implements UpstreamProvider, storing a fixed set of URLs, and
// caches parsed URLs and any error that occurred.
type fixedSet struct {
	urlStrs []string
	pick    func(urls []*url.URL) *url.URL

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
	return ru.pick(ru.urls), nil
}

// All returns all upsteams.
func (ru *fixedSet) All(req *http.Request) ([]*url.URL, error) {
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
			ru.err = err
			return
		}
		urls[i] = u
	}
	ru.urls = urls
}
