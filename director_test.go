package locus

import (
	"net/http"
	"testing"
)

func TestPathForwarding(t *testing.T) {
	dir := Director{
		UpstreamProvider: Single("https://google.com:4000"),
		PathPrefix:       "/search",
	}

	req, _ := dir.Direct(mustReq("http://my.mirror.com/search/Byzantine–Bulgarian_wars"))

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

func TestPath(t *testing.T) {
	dir := Director{
		UpstreamProvider: Single("https://en.wikipedia.org/wiki/"),
		PathPrefix:       "/stuff",
	}

	req, _ := dir.Direct(mustReq("http://my.mirror.com/stuff/Byzantine–Bulgarian_wars"))

	actual := req.URL.String()
	expected := "https://en.wikipedia.org/wiki/Byzantine%E2%80%93Bulgarian_wars"
	if actual != expected {
		t.Errorf("Expected URL to be '%s', was %s", expected, actual)
	}
}

func TestPathNoTrailingSlash(t *testing.T) {
	dir := Director{
		UpstreamProvider: Single("http://www.bbc.com/news"),
		PathPrefix:       "/",
	}

	req, _ := dir.Direct(mustReq("http://localhost:1234/"))

	actual := req.URL.String()
	expected := "http://www.bbc.com/news"
	if actual != expected {
		t.Errorf("Expected URL to be '%s', was %s", expected, actual)
	}
}

func TestHeaders(t *testing.T) {
	dir := Director{
		UpstreamProvider: Single("https://en.wikipedia.org/wiki/"),
		PathPrefix:       "/stuff",
	}
	dir.StripHeader("Cookie")
	dir.SetHeader("Referer", "https://en.wikipedia.org/wiki/Main_Page")

	req := mustReq("http://my.mirror.com/stuff/Byzantine–Bulgarian_wars")
	req.AddCookie(&http.Cookie{Name: "foo", Value: "bar"})
	req.Header.Set("Referer", "http://mysite.com")

	proxyReq, _ := dir.Direct(req)

	if c, ok := proxyReq.Header["Cookie"]; ok {
		t.Errorf("Expected Cookie to be stripped, was %s", c)
	}

	if h := proxyReq.Header["Referer"]; len(h) != 1 || h[0] != "https://en.wikipedia.org/wiki/Main_Page" {
		t.Errorf("Expected Referer to be overwritten, was %s", proxyReq.Header["Referer"])
	}
}
