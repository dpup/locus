package locus

import (
	"net/http"
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

func checkError(t *testing.T, err error, str string) {
	if err != nil {
		t.Errorf("unexpected error: %s: %s", str, err)
	}
}
