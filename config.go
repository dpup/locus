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

// Bind uses an URL to define the host:port/path?query components to match on.
//
// If present, Host and Port will be exact matches, Path is prefix matched.
// Query params are exact match, but not exclusive. Scheme is ignored. A URL
// with a host and no no port will match all ports.
//
// See Matcher_test.go for examples.
func (c *Config) Bind(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	c.BindHost(u.Host)
	c.BindLocation(u.RequestURI())
	return nil
}

// BindLocation sets the path and query (request URI) portion that should be
// matched.
func (c *Config) BindLocation(requestURI string) {
	path, _ := c.Matcher.BindLocation(requestURI)
	c.PathPrefix = path
}

// Upstream specifies an UpstreamProvider to use when finding the destination
// server.
//
// `Single("http://dest.com")` can be used to route requests to a single
// upstream server. `Random(urls)` or `RoundRobin(urls)` can be used to choose
// from a fixed set of servers. Other implementations exist.
func (c *Config) Upstream(u UpstreamProvider) {
	c.Director.UpstreamProvider = u
}
