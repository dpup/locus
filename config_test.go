package locus

import (
	"net/http"
	"testing"
)

func TestTransformWithPathForwarding(t *testing.T) {
	cfg := Config{}
	cfg.Upstream(Single("https://google.com:4000"))
	cfg.Bind("http://my.mirror.com/search/")

	req := mustReq("http://my.mirror.com/search/Byzantine–Bulgarian_wars")

	cfg.Transform(req)

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

func TestTransformWithPath(t *testing.T) {
	cfg := Config{}
	cfg.Upstream(Single("https://en.wikipedia.org/wiki/"))
	cfg.Bind("http://my.mirror.com/stuff/")

	req := mustReq("http://my.mirror.com/stuff/Byzantine–Bulgarian_wars")

	cfg.Transform(req)

	actual := req.URL.String()
	expected := "https://en.wikipedia.org/wiki/Byzantine%E2%80%93Bulgarian_wars"
	if actual != expected {
		t.Errorf("Expected URL to be '%s', was %s", expected, actual)
	}
}

func TestTransformWithPathNoTrailingSlash(t *testing.T) {
	cfg := Config{}
	cfg.Upstream(Single("http://www.bbc.com/news"))
	cfg.Bind("/")

	req := mustReq("http://localhost:1234/")
	cfg.Transform(req)

	actual := req.URL.String()
	expected := "http://www.bbc.com/news"
	if actual != expected {
		t.Errorf("Expected URL to be '%s', was %s", expected, actual)
	}
}

func TestTransformWithHeaders(t *testing.T) {
	cfg := Config{}
	cfg.Upstream(Single("https://en.wikipedia.org/wiki/"))
	cfg.Bind("http://my.mirror.com/stuff/")
	cfg.StripHeader("Cookie")
	cfg.SetHeader("Referer", "https://en.wikipedia.org/wiki/Main_Page")

	req := mustReq("http://my.mirror.com/stuff/Byzantine–Bulgarian_wars")
	req.AddCookie(&http.Cookie{Name: "foo", Value: "bar"})
	req.Header.Set("Referer", "http://mysite.com")

	cfg.Transform(req)

	if c, ok := req.Header["Cookie"]; ok {
		t.Errorf("Expected Cookie to be stripped, was %s", c)
	}

	if h := req.Header["Referer"]; len(h) != 1 || h[0] != "https://en.wikipedia.org/wiki/Main_Page" {
		t.Errorf("Expected Referer to be overwritten, was %s", req.Header["Referer"])
	}
}
