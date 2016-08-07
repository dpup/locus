package locus

import (
	"net/http"
	"net/url"
	"strings"
)

// The Matcher interface is used to determine if a config matches an incoming
// request.
type Matcher interface {
	Matches(req *http.Request) bool
}

type urlMatcher struct {
	url          *url.URL
	preprocessed bool
	host         string
	wild         bool
	port         string
	query        url.Values
}

func (um *urlMatcher) String() string {
	return um.url.String()
}

func (um *urlMatcher) Matches(req *http.Request) bool {
	ok, _ := um.MatchWithReason(req.URL)
	return ok
}

func (um *urlMatcher) MatchWithReason(u *url.URL) (bool, string) {
	um.preprocess()

	if um.url.Scheme != "" && um.url.Scheme != u.Scheme {
		return false, "scheme mismatch"
	}
	if um.url.Host != "" && !um.matchHost(u) {
		return false, "host mismatch"
	}
	if um.url.Path != "" && !strings.HasPrefix(u.Path, um.url.Path) {
		return false, "path prefix mismatch"
	}
	if um.url.RawQuery != "" && !um.matchQuery(u) {
		return false, "query mismatch"
	}
	return true, "match"
}

func (um *urlMatcher) preprocess() {
	if !um.preprocessed {
		um.host, um.port = splitHost(um.url)
		if um.host != "" && um.host[:1] == "*" {
			um.wild = true
			um.host = um.host[1:]
		}
		um.query = um.url.Query()
		um.preprocessed = true
	}
}

func (um *urlMatcher) matchHost(u *url.URL) bool {
	host, port := splitHost(u)
	return (um.host == "" || um.host == host || (um.wild && strings.HasSuffix(host, um.host))) &&
		(um.port == "" || um.port == port)
}

func (um *urlMatcher) matchQuery(u *url.URL) bool {
	query := u.Query()
	for k, v := range um.query {
		if query.Get(k) != v[0] {
			return false
		}
	}
	return true
}

func splitHost(u *url.URL) (host, port string) {
	parts := strings.Split(u.Host, ":")
	host = parts[0]
	if len(parts) == 2 {
		port = parts[1]
	} else if u.Scheme == "https" {
		port = "443"
	} else if u.Scheme == "http" {
		port = "80"
	}
	return
}
