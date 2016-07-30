package locus

import (
	"math/rand"
	"net/http"
	"net/url"
	"strings"
)

// TODO: hook up httputil.ReverseProxy with custom Director.
// https://golang.org/src/net/http/httputil/reverseproxy.go?s=3769:3844#L130

// Config is...
type Config struct {
	// RequestMatcher is used to determine whether a config matches an incoming
	// request and should be used to configure the proxied request.
	RequestMatcher RequestMatcher

	// UpstreamProvider is used to fetch a list of candidate upstreams to proxy
	// the request to.
	UpstreamProvider UpstreamProvider

	// PathPrefix will be stripped from the incoming request path, iff the
	// upstream specifies a path in its URL.
	PathPrefix string

	stripHeaders []string
	setHeaders   map[string]string
	addHeaders   map[string][]string

	// TODO: Upstream retry settings.
	// TODO: Upstream failover settings.
	// TODO: Upstream selection settings.
	//
	// TODO: Allow response transformations.
}

// NewConfig returns an empty config instance.
func NewConfig() *Config {
	return *Config{}
}

// Transform applies a config to an HTTP request, satisifies the same signature
// as httputil.ReverseProxy.Director.
//
// The UpstreamProvider is used to get a list of candidate upstreams, for now a
// random one is chosen. The upstream is then used to set scheme and host on the
// URL.
//
// If the upstream path is empty, the path is left unaltered. If the upststream
// path is non empty, e.g. '/' or '/some/prefix/', then the proxied request's
// path is set to the upstream path joined with a trimmed request path. For
// default RequestMatcher the required path prefix is stripped from the proxied
// request.
//
// Examples 1: Pathless upstream proxies entire request path.
//
//     match     = http://abc.com/def
//     upstream  = http://upstream.com
//     request   = http://abc.com/def/ghi
//     proxied   = http://upstream.com/def/ghi
//
// Examples 2: Upstream with trailing slash strips matched prefix.
//
//     match     = http://abc.com/def
//     upstream  = http://upstream.com/
//     request   = http://abc.com/def/ghi
//     proxied   = http://upstream.com/ghi
//
// Examples 3: Upstream with path, strips matched prefix and concats remainder.
//
//     match     = http://abc.com/def
//     upstream  = http://upstream.com/xyz
//     request   = http://abc.com/def/ghi
//     proxied   = http://upstream.com/xyz/ghi
//
func (c *Config) Transform(req *http.Request) {
	upstreams := c.UpstreamProvider.Upstreams()
	upstream := upstreams[rand.Intn(len(upstreams))]

	// Update destination.
	req.URL.Scheme = upstream.Scheme
	req.URL.Host = upstream.Host

	if upstream.Path != "" {
		pathSuffix := strings.TrimPrefix(req.URL.Path, c.PathPrefix)
		req.URL.Path = singleJoiningSlash(upstream.Path, pathSuffix)
	}

	// Strip, set and add headers.
	for _, h := range c.stripHeaders {
		delete(req.Header, h)
	}
	for k, v := range c.setHeaders {
		req.Header[k] = []string{v}
	}
	for k, v := range c.addHeaders {
		if _, ok := req.Header[k]; !ok {
			req.Header[k] = []string{}
		}
		req.Header[k] = append(req.Header[k], v...)
	}
}

// Match configures the config to match a URL. Scheme, Host, Port should be
// exact matches, Path is prefix matched. Ports 80 and 443 are implied.
func (c *Config) Match(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	c.RequestMatcher = &urlMatcher{url: u}
	c.PathPrefix = u.Path
	return nil
}

// Upstream configures the revproxy to use a fixed destination.
func (c *Config) Upstream(u string) error {
	upstreams := []string{u}
	return c.Upstreams(upstreams)
}

// Upstreams configures the revproxy to select from a fixed set of destinations.
func (c *Config) Upstreams(urlStrings []string) error {
	upstreams := make([]*url.URL, len(urlStrings))
	for i, urlStr := range urlStrings {
		u, err := url.Parse(urlStr)
		if err != nil {
			return err
		}
		upstreams[i] = u
	}
	fn := UpstreamProviderFn(func() []*url.URL {
		return upstreams
	})
	c.UpstreamProvider = &fn
	return nil
}

// AddHeader specifies a header to add to the proxied request.
func (c *Config) AddHeader(key, value string) {
	if c.addHeaders == nil {
		c.addHeaders = map[string][]string{}
	}
	key = http.CanonicalHeaderKey(key)
	if _, ok := c.addHeaders[key]; !ok {
		c.addHeaders[key] = []string{}
	}
	c.addHeaders[key] = append(c.addHeaders[key], value)
}

// SetHeader specifies a header to set on the proxied request, overriding any
// value that already exists.
func (c *Config) SetHeader(key, value string) {
	if c.setHeaders == nil {
		c.setHeaders = map[string]string{}
	}
	key = http.CanonicalHeaderKey(key)
	c.setHeaders[key] = value
}

// StripHeader specifices a header to be removed from the proxied request.
func (c *Config) StripHeader(key string) {
	key = http.CanonicalHeaderKey(key)
	c.stripHeaders = append(c.stripHeaders, key)
}

// From https://golang.org/src/net/http/httputil/reverseproxy.go#L63
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}