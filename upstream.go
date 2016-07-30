package locus

import (
	"math/rand"
	"net/url"
)

// UpstreamProvider is...
type UpstreamProvider interface {
	Get() (*url.URL, error)
}

// SingleUpstream is an adaptor to allow a url.URL to act as a SingleUpstream.
// For example:
//     cfg.Upstream(SingleUpstream("http://test.com"))
type SingleUpstream string

// Get simply returns the URL.
func (urlStr SingleUpstream) Get() (*url.URL, error) {
	return url.Parse(string(urlStr)) // TODO: memoize?
}

// RndUpstream is an adaptor which picks a random URL from a fixed list of URLs.
// For example:
//     cfg.Upstream(RndUpstream([]string{
//       "back-1.test.com",
//       "back-2.test.com",
//       "back-3.test.com",
//     }))
type RndUpstream []string

// Get returns a random URL.
func (urlStrs RndUpstream) Get() (*url.URL, error) {
	urlStr := urlStrs[rand.Intn(len(urlStrs))]
	return url.Parse(urlStr) // TODO: memoize?
}
