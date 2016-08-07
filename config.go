package locus

import (
	"net/url"
)

// Config specifies what requests to handle, how to direct the request, and
// how to transform the response.
type Config struct {
	// Matcher specifies what requests this config should operate on.
	Matcher

	// Director specifies how to proxy/redirect requests.
	Director

	// User visible name for the config, used in debug pages and logs.
	Name string

	// Redirect specfied a HTTP status code that should be issued along with a
	// Location header. Should one of be 301, 302, 307.
	Redirect int
}

// Bind configures this config to target the provided URL.
//
// If present, Scheme, Host, Port will be exact matches, Path is prefix matched.
// Query params are exact match, but not exclusive. Ports 80 and 443 are implied
// if a scheme is present without explicit port. A URL with a host and no schemeor port will match all ports.
//
// See Matcher_test.go for examples.
func (c *Config) Bind(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	c.Matcher = &urlMatcher{url: u}
	c.PathPrefix = u.Path
	return nil
}

// Upstream specifies an UpstreamProvider to use when finding the destination
// server.
//
// `Single("http://dest.com")` can be used to route requests to a single
// upstream server. `Random(urls)` or `RoundRobin(urls)` can be used to choose
// from a fixed set of servers. Other implementations exist.
func (c *Config) Upstream(u UpstreamProvider) {
	c.UpstreamProvider = u
}
