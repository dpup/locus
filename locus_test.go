package locus

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"testing"
)

func mustParseURL(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return u
}

func mustReq(rawurl string) *http.Request {
	r, err := http.NewRequest("GET", rawurl, nil)
	if err != nil {
		panic(err)
	}
	return r
}

func mustDump(req *http.Request) string {
	d, err := httputil.DumpRequest(req, false)
	if err != nil {
		panic(err)
	}
	return "Parsed URL: " + req.URL.String() + "\n" +
		"Dump: " + string(d)
}

func checkError(t *testing.T, err error, str string) {
	if err != nil {
		t.Fatalf("unexpected error: %s: %s", str, err)
	}
}
