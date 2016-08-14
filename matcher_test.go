package locus

import (
	"testing"
)

func TestUrlMatcher(t *testing.T) {
	var urlTests = []struct {
		matchHost string
		matchPath string
		requrl    string
		expected  bool
	}{
		// Host only binding.
		{"test.com", "", "//test.com", true},
		{"test.com", "", "http://test.com", true},
		{"test.com", "", "https://test.com", true},
		{"test.com", "", "ftp://test.com", true},
		{"test.com", "", "http://test.com:5000", true},
		{"test.com", "", "http://test.com/foobar", true},
		{"test.com", "", "http://test.com/foobar/bazbar", true},
		{"test.com", "", "https://test.com/foobar", true},
		{"test.com", "", "https://www.test.com", false},
		{"test.com", "", "www.test.com", false},

		// Wildcard host binding.
		{".test.com", "", "http://test.com", false}, // Should this be true?
		{".test.com", "", "http://notmytest.com", false},
		{".test.com", "", "http://www.test.com", true},
		{".test.com", "", "http://about.test.com", true},
		{".test.com", "", "http://one.two.three.test.com", true},

		// Full host and port binding.
		{"test.com:5000", "", "http://test.com/foo", false},
		{"test.com:5000", "", "http://test.com:5000/foo", true},

		// Host and path binding.
		{"test.com", "/foo", "http://test.com/foo", true},
		{"test.com", "/foo", "http://test.com/foo/", true},
		{"test.com", "/foo", "http://test.com/foo/bar", true},
		{"test.com", "/foo", "http://test.com/baz", false},

		// Path only binding.
		{"", "/foo", "http://test.com/foo", true},
		{"", "/foo", "http://google.com/foo/bar", true},
		{"", "/foo", "http://google.com/baz/foo/bar", false},

		// Port only binding.
		{":5000", "", "http://test.com:5000/foo", true},
		{":5000", "", "https://google.com:5000/foo/bar", true},
		{":5000", "", "http://google.com/baz/foo/bar", false},

		// Port 80 is implied for HTTP.
		{"test.com", "", "http://test.com:80/foo", true},
		{"test.com:80", "", "http://test.com/foo", true},
		{"test.com:80", "", "http://test.com:80/foo", true},
		{"test.com:80", "", "http://test.com:5000/foo", false},

		// Port 443 is implied for HTTPS.
		{"test.com:443", "", "https://test.com/foo", true},
		{"test.com:443", "", "https://test.com:443/foo", true},
		{"test.com:443", "", "http://test.com:443/foo", true},
		{"test.com:443", "", "http://test.com/foo", false},
		{"test.com:443", "", "https://test.com:5000/foo", false},

		// Query param binding.
		{"", "?staging=true", "http://test.com/?staging=true", true},
		{"", "?staging=true", "http://test.com/?staging=true&debug=true", true},
		{"", "?staging=true", "http://test.com/?staging=false", false},
		{"", "?staging=true", "http://test.com/?staging=false&staging=true", false},
		{"", "?staging=true", "http://test.com/?staging=1", false},
		{"", "?staging=true", "http://test.com/", false},
		{"", "?lang=en&country=us", "http://test.com/?lang=en&country=us", true},
		{"", "?lang=en&country=us", "http://test.com/?country=us&lang=en", true},
		{"", "?lang=en&country=us", "http://test.com/?lang=en", false},
		{"", "?lang=en&country=us", "http://test.com/?country=us", false},

		// Incoming URLs without '//' are hostless, test.com is actually the path.
		{"test.com", "", "test.com", false},
	}

	for _, tt := range urlTests {
		req := mustReq(tt.requrl)
		um := NewMatcher(tt.matchHost, tt.matchPath)
		actual, reason := um.Match(req)
		if actual != tt.expected {
			t.Errorf("matching '%s' against '%s' => %v, want %v (%s)", tt.requrl, um, actual, tt.expected, reason)
		}
	}
}

// Per RFC 2616, Section 5.1.2, most request URLs will only be path+query. The
// above test uses fullformed URLs. This test briefly ensures the Host header
// is used when the URL's host is empty.
func TestHostless(t *testing.T) {
	req := mustReq("http://www.test.com/foo/bar/baz")
	req.URL.Host = ""
	req.URL.Scheme = ""

	um1 := NewMatcher("www.test.com", "/foo")
	if ok, reason := um1.Match(req); !ok {
		t.Errorf("Expected match, but got reason '%s'", reason)
	}

	um2 := NewMatcher("poop.test.com", "/foo/bar/baz")
	if ok, _ := um2.Match(req); ok {
		t.Errorf("Didn't expected a match")
	}
}
