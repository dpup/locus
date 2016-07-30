package locus

import (
	"net/http"
	"testing"
)

func TestTransformWithPathForwarding(t *testing.T) {
	cfg := Config{}
	cfg.Upstream("https://google.com:4000")
	cfg.Match("http://my.mirror.com/search/")

	req := mustReq("http://my.mirror.com/search/Byzantine–Bulgarian_wars")

	cfg.Transform(&req)

	if req.URL.Scheme != "https" {
		t.Errorf("Expected scheme to be 'https', was %s", req.URL.Scheme)
	}

	if req.URL.Host != "google.com:4000" {
		t.Errorf("Expected host to be 'google.com:4000', was %s", req.URL.Host)
	}

	if req.URL.Path != "/search/Byzantine–Bulgarian_wars" {
		t.Errorf("Expected path to be '/search/Byzantine–Bulgarian_wars', was %s", req.URL.Path)
	}
}

func TestTransformWithPathModification(t *testing.T) {
	cfg := Config{}
	cfg.Upstream("https://en.wikipedia.org/wiki/")
	cfg.Match("http://my.mirror.com/stuff/")

	req := mustReq("http://my.mirror.com/stuff/Byzantine–Bulgarian_wars")

	cfg.Transform(&req)

	if req.URL.Scheme != "https" {
		t.Errorf("Expected scheme to be 'https', was %s", req.URL.Scheme)
	}

	if req.URL.Host != "en.wikipedia.org" {
		t.Errorf("Expected host to be 'en.wikipedia.org', was %s", req.URL.Host)
	}

	if req.URL.Path != "/wiki/Byzantine–Bulgarian_wars" {
		t.Errorf("Expected path to be '/wiki/Byzantine–Bulgarian_wars', was %s", req.URL.Path)
	}
}

func TestTransformWithHeaders(t *testing.T) {
	cfg := Config{}
	cfg.Upstream("https://en.wikipedia.org/wiki/")
	cfg.Match("http://my.mirror.com/stuff/")
	cfg.StripHeader("Cookie")
	cfg.SetHeader("Referer", "https://en.wikipedia.org/wiki/Main_Page")

	req := mustReq("http://my.mirror.com/stuff/Byzantine–Bulgarian_wars")
	req.AddCookie(&http.Cookie{Name: "foo", Value: "bar"})
	req.Header.Set("Referer", "http://mysite.com")

	cfg.Transform(&req)

	if c, ok := req.Header["Cookie"]; ok {
		t.Errorf("Expected Cookie to be stripped, was %s", c)
	}

	if h := req.Header["Referer"]; len(h) != 1 || h[0] != "https://en.wikipedia.org/wiki/Main_Page" {
		t.Errorf("Expected Referer to be overwritten, was %s", req.Header["Referer"])
	}
}
