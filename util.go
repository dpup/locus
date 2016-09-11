package locus

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func copyRequest(req *http.Request) *http.Request {
	cr := new(http.Request)
	*cr = *req
	cr.URL = &url.URL{}
	*cr.URL = *req.URL
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

func newLogger(filename string) (*log.Logger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("unable to create logger: %v", err)
	}
	return log.New(io.MultiWriter(os.Stderr, file), "", log.Ldate|log.Ltime), nil
}

func remoteAddr(req *http.Request) string {
	if ff := req.Header.Get("X-Forwarded-For"); ff != "" {
		return ff
	}
	return req.RemoteAddr
}
