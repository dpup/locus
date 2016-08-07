package locus

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	cfgs, globals, err := loadConfigFromYAML([]byte(SampleYAMLConfig))

	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if globals.Port != 5556 {
		t.Errorf("Expected port 5556, was %d", globals.Port)
	}

	if globals.ReadTimeout != 10*time.Second {
		t.Errorf("Expected read timeout to be 10s, was %s", globals.ReadTimeout)
	}

	if globals.WriteTimeout != 20*time.Second {
		t.Errorf("Expected write timeout to be 20s, was %s", globals.WriteTimeout)
	}

	about := cfgs[0]
	search := cfgs[1]
	fallthru := cfgs[2]
	redirect := cfgs[3]

	// Verify the first site has a single URL upstream.
	actual1, err := about.UpstreamProvider.All()
	expected1 := []*url.URL{mustParseURL("http://about-1.mysite.com")}
	checkError(t, err, "fetching 'about' upstreams")
	if !reflect.DeepEqual(actual1, expected1) {
		t.Errorf("Unexpected upstreams, expected '%s' was '%s'", expected1, actual1)
	}

	// Verify the second site has a fixed set of URLs.
	actual2, err := search.UpstreamProvider.All()
	expected2 := []*url.URL{
		mustParseURL("http://search-1.mysite.com"),
		mustParseURL("http://search-2.mysite.com"),
		mustParseURL("http://search-3.mysite.com"),
	}
	checkError(t, err, "fetching 'search' upstreams")
	if !reflect.DeepEqual(actual2, expected2) {
		t.Errorf("Unexpected upstreams, expected '%s' was '%s'", expected2, actual2)
	}

	// Verify the third site uses DNS.
	if ds, ok := fallthru.UpstreamProvider.(*DNSSet); ok {
		actual3, err := ds.All()
		expected3 := []*url.URL{
			mustParseURL("http://192.168.0.0:4000/2016/mysite/"),
			mustParseURL("http://192.168.0.1:4000/2016/mysite/"),
			mustParseURL("http://192.168.0.2:4000/2016/mysite/"),
			mustParseURL("http://192.168.0.3:4000/2016/mysite/"),
		}
		checkError(t, err, "fetching 'fallthru' upstreams")
		if !reflect.DeepEqual(actual3, expected3) {
			t.Errorf("Unexpected upstreams, expected '%s' was '%s'", expected3, actual3)
		}
		if !ds.RoundRobin {
			t.Errorf("Expected RoundRobin to be true, was false")
		}
		if !ds.AllowStale {
			t.Errorf("Expected AllowStale to be true, was false")
		}
		if ds.TTL != time.Minute*5 {
			t.Errorf("Expected TTL to be 5m, was %s", ds.TTL)
		}
	} else {
		t.Errorf("Expected fallthru to have DNS provider, was %v", fallthru)
	}

	// Check that global AddHeader set.
	if v, ok := about.addHeaders["X-Proxied-For"]; !ok || !reflect.DeepEqual(v, []string{"Locus"}) {
		t.Errorf("Unexpected global header for 'X-Proxied-For', was '%v'", v)
	}

	// Check local SetHeader is set.
	if v, ok := about.setHeaders["Accept-Language"]; !ok || v != "en-US" {
		t.Errorf("Unexpected local header for 'Accept-Language', was '%v'", v)
	}

	// Check local StripHeader is set.
	expected := []string{"Cookie", "User-Agent"}
	if !reflect.DeepEqual(about.stripHeaders, expected) {
		t.Errorf("Unexpected strip header, wanted %v was '%v'", expected, about.stripHeaders)
	}

	if redirect.Redirect != http.StatusMovedPermanently {
		t.Errorf("Unexpected redirect, wanted %d was %d", http.StatusMovedPermanently, redirect.Redirect)
	}
}
