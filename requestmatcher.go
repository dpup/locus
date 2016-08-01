package locus

import (
	"net/http"
	"net/url"
	"strings"
)

// The RequestMatcher interface is used to determine if a config matches an
// incoming request.
type RequestMatcher interface {
	Matches(req *http.Request) bool
}

// RequestMatcherFn is an adaptor to allow a function to expose the
// RequestMatcher interface.
type RequestMatcherFn func(req *http.Request) bool

// Matches calls the function and returns the result.
func (fn RequestMatcherFn) Matches(req *http.Request) bool {
	return fn(req)
}

// MatchAll implements RequestMatcher interface and matches all requests.
var MatchAll = RequestMatcherFn(func(req *http.Request) bool {
	return true
})

type urlMatcher struct {
	url          *url.URL
	preprocessed bool
	host         string
	port         string
}

func (um *urlMatcher) Matches(req *http.Request) bool {
	ok, _ := um.MatchWithReason(req)
	return ok
}

func (um *urlMatcher) MatchWithReason(req *http.Request) (bool, string) {
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
}

func (um *urlMatcher) matchHost(req *http.Request) bool {
	if !um.preprocessed {
		um.host, um.port = splitHost(um.url)
		um.preprocessed = true
	}
	host, port := splitHost(req.URL)
	return (um.host == "" || um.host == host) && (um.port == "" || um.port == port)
}

func splitHost(url *url.URL) (host, port string) {
	parts := strings.Split(url.Host, ":")
	host = parts[0]
	if len(parts) == 2 {
		port = parts[1]
	} else if url.Scheme == "https" {
		port = "443"
	} else if url.Scheme == "http" {
		port = "80"
	}
	return
}
