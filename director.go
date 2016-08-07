package locus

import (
	"net/http"
	"strings"
)

// Director specifies how to direct a request to an upstream backend.
type Director struct {
	// PathPrefix will be stripped from the incoming request path, iff the
	// upstream specifies a path in its URL. When using Config.Bind() it is set to
	// the path provided in the URL to be matched.
	PathPrefix string

	// UpstreamProvider is used to fetch a list of candidate upstreams to proxy
	// the request to.
	UpstreamProvider UpstreamProvider

	stripHeaders []string
	setHeaders   map[string]string
	addHeaders   map[string][]string
}

// Direct mutates a HTTP request, for proxying to an upstream server.
//
// By default, the Host header is not set to the upstream's host, as it is
// common for upstreams to be IPs and to want the Host from the original
// request. Use SetHeader("Host", "foo.com") if you desire alternate behavior.
//
// The UpstreamProvider is used to get a list of candidate upstreams, for now a
// random one is chosen. The upstream is then used to set scheme and host on the
// URL.
//
// If the upstream path is empty, the path is left unaltered. If the upststream
// path is non empty, e.g. '/' or '/some/prefix/', then the proxied request's
// path is set to the upstream path joined with a trimmed request path. For
// default Matcher the required path prefix is stripped from the proxied
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
func (d *Director) Direct(req *http.Request) (*http.Request, error) {
	upstream, err := d.UpstreamProvider.Get(req)
	if err != nil {
		return nil, err
	}

	req = copyRequest(req)

	// Update destination.
	req.URL.Scheme = upstream.Scheme
	req.URL.Host = upstream.Host

	if upstream.Path != "" {
		pathSuffix := strings.TrimPrefix(req.URL.Path, d.PathPrefix)
		if pathSuffix == "" {
			req.URL.Path = upstream.Path
		} else {
			req.URL.Path = singleJoiningSlash(upstream.Path, pathSuffix)
		}
	}

	// Strip, set and add headers.
	for _, h := range d.stripHeaders {
		delete(req.Header, h)
	}
	for k, v := range d.setHeaders {
		req.Header[k] = []string{v}
		if k == "Host" {
			req.Host = v
		}
	}
	for k, v := range d.addHeaders {
		if _, ok := req.Header[k]; !ok {
			req.Header[k] = []string{}
		}
		req.Header[k] = append(req.Header[k], v...)
	}

	return req, nil
}

// AddHeader specifies a header to add to the proxied request.
func (d *Director) AddHeader(key, value string) {
	if d.addHeaders == nil {
		d.addHeaders = map[string][]string{}
	}
	key = http.CanonicalHeaderKey(key)
	if _, ok := d.addHeaders[key]; !ok {
		d.addHeaders[key] = []string{}
	}
	d.addHeaders[key] = append(d.addHeaders[key], value)
}

// SetHeader specifies a header to set on the proxied request, overriding any
// value that already exists.
func (d *Director) SetHeader(key, value string) {
	if d.setHeaders == nil {
		d.setHeaders = map[string]string{}
	}
	key = http.CanonicalHeaderKey(key)
	d.setHeaders[key] = value
}

// StripHeader specifices a header to be removed from the proxied request.
func (d *Director) StripHeader(key string) {
	key = http.CanonicalHeaderKey(key)
	d.stripHeaders = append(d.stripHeaders, key)
}
