package locus

import (
	"net/http"
	"strings"
)

func copyRequest(req *http.Request) *http.Request {
	cr := new(http.Request)
	*cr = *req
	cr.Header = make(http.Header)
	copyHeader(cr.Header, req.Header)
	return cr
}

// From https://golang.org/src/net/http/httputil/reverseproxy.go
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// From https://golang.org/src/net/http/httputil/reverseproxy.go
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
