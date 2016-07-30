package locus

import (
	"net/url"
)

// UpstreamProvider is...
type UpstreamProvider interface {
	Upstreams() []*url.URL
}

// UpstreamProviderFn is an adaptor to allow a function to expose the
// UpstreamProvider interface
type UpstreamProviderFn func() []*url.URL

// Upstreams calls the UpstreamProviderFn and returns the result.
func (fn UpstreamProviderFn) Upstreams() []*url.URL {
	return fn()
}
