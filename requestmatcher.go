package locus

import (
	"net/http"
	"net/url"
	"strings"
)

// RequestMatcher is...
type RequestMatcher interface {
	Matches(req http.Request) bool
}

// RequestMatcherFn is an adaptor to allow a function to expose the
// RequestMatcher interface.
type RequestMatcherFn func(req http.Request) bool

// Matches calls the function and returns the result.
func (fn RequestMatcherFn) Matches(req http.Request) bool {
	return fn(req)
}

// MatchAll implements RequestMatcher interface and matches all requests.
var MatchAll = RequestMatcherFn(func(req http.Request) bool {
	return true
})

type urlMatcher struct {
	url      *url.URL
	hostPort string
}

func (um *urlMatcher) Matches(req http.Request) bool {
	ok, _ := um.MatchWithReason(req)
	return ok
}

func (um *urlMatcher) MatchWithReason(req http.Request) (bool, string) {
	if um.url.Scheme != "" && um.url.Scheme != req.URL.Scheme {
		return false, "scheme mismatch"
	}
	if um.url.Host != "" && !um.matchHost(req) {
		return false, "host mismatch"
	}
	if um.url.Path != "" && !strings.HasPrefix(req.URL.Path, um.url.Path) {
		return false, "path prefix mismatch"
	}
	return true, "match"

	// TODO: Add query param matching, e.g. foo.com?staging=true.
}

func (um *urlMatcher) matchHost(req http.Request) bool {
	// TODO: Add wildcard domain matching.

	if um.hostPort == "" {
		um.hostPort = um.url.Host
		if !strings.Contains(um.hostPort, ":") {
			if um.url.Scheme == "http" {
				um.hostPort += ":80"
			} else if um.url.Scheme == "https" {
				um.hostPort += ":443"
			}
		}
	}

	if req.URL.Host == um.url.Host || req.URL.Host == um.hostPort {
		return true
	}
	if req.URL.Scheme == "http" && req.URL.Host+":80" == um.hostPort {
		return true
	}
	if req.URL.Scheme == "https" && req.URL.Host+":443" == um.hostPort {
		return true
	}
	return false
}
