package locus

import (
	"net/http"
	"net/url"
	"strings"
)

// Matcher is used to match incoming requests
type Matcher struct {
	host     string
	port     string
	wild     bool
	path     string
	hasQuery bool
	query    url.Values
}

// NewMatcher constructs a matcher from a hostPort and requestURI.
func NewMatcher(hostPort string, requestURI string) *Matcher {
	um := &Matcher{}
	um.BindHost(hostPort)
	um.BindLocation(requestURI)
	return um
}

// BindHost sets which host and port to match on, if either host or port are
// blank then they will match any value.
// Example inputs include: "www.test.com", "test.com:5000", ":80".
func (um *Matcher) BindHost(hostPort string) (string, string) {
	um.host, um.port = splitHost(hostPort)

	// If host starts with "." it's a wildcard match.
	if um.host != "" && um.host[:1] == "." {
		um.wild = true
	} else {
		um.wild = false
	}
	return um.host, um.port
}

// BindLocation sets the path and query (request URI) portion that should be
// matched. Path will prefix match, all query params will be matched.
func (um *Matcher) BindLocation(requestURI string) (string, url.Values) {
	if requestURI == "" {
		um.path = ""
		um.query = nil
	} else {
		if requestURI[:1] == "?" {
			// Make the API a bit more intuitive, don't require people to bind "/?foo"
			requestURI = "/" + requestURI
		}

		// requestURI should be path?query, but if it doesn't parse take it as path.
		if u, err := url.ParseRequestURI(requestURI); err == nil {
			um.path = u.Path
			if u.RawQuery != "" {
				um.query = u.Query()
				um.hasQuery = true
			} else {
				um.hasQuery = false
				um.query = nil
			}
		} else {
			um.path = requestURI
			um.hasQuery = false
			um.query = nil
		}
	}
	return um.path, um.query
}

func (um Matcher) String() string {
	str := ""
	if um.host != "" {
		str += um.host
	}
	if um.port != "" {
		str += ":" + um.port
	}
	if um.path != "" {
		str += um.path
	}
	if um.hasQuery {
		str += um.query.Encode()
	}
	return str
}

// Match returns true if an inbound request satisfies all the requirements of
// the matcher.
func (um *Matcher) Match(req *http.Request) (bool, string) {
	// Per RFC 2616 most request URLs will only include path+query. For purpose of
	// matching we rely on the host header.
	host, port := splitHost(req.Host)

	// TODO(dan): should this fallback on req.URI.Host?

	if um.host != "" && !um.matchHost(host) {
		return false, "host mismatch"
	}
	if um.port != "" && !um.matchPort(port, req.URL.Scheme) {
		return false, "port mismatch"
	}
	if um.path != "" && !strings.HasPrefix(req.URL.Path, um.path) {
		return false, "path prefix mismatch"
	}
	if um.hasQuery && !um.matchQuery(req.URL) {
		return false, "query mismatch"
	}
	return true, "match"
}

func (um *Matcher) matchHost(host string) bool {
	if um.wild {
		return strings.HasSuffix(host, um.host)
	}
	return host == um.host
}

func (um *Matcher) matchPort(port string, scheme string) bool {
	if um.port == port {
		// Direct match.
		return true
	} else if um.port == "80" && port == "" && scheme == "http" {
		// For fully formed req URLs, allow http to imply port 80.
		return true
	} else if um.port == "443" && port == "" && scheme == "https" {
		// For fully formed req URLs, allow https to imply port 443.
		return true
	}
	return false
}

func (um *Matcher) matchQuery(u *url.URL) bool {
	query := u.Query()
	for k, v := range um.query {
		if query.Get(k) != v[0] {
			return false
		}
	}
	return true
}

func splitHost(hostPort string) (host, port string) {
	parts := strings.Split(hostPort, ":")
	host = parts[0]
	if len(parts) == 2 {
		port = parts[1]
	}
	return
}
