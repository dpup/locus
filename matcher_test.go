package locus

import (
	"testing"
)

func TestUrlMatcher(t *testing.T) {
	var urlTests = []struct {
		matchurl string
		requrl   string
		expected bool
	}{
		// Host only binding.
		{"//test.com", "//test.com", true},
		{"//test.com", "http://test.com", true},
		{"//test.com", "https://test.com", true},
		{"//test.com", "ftp://test.com", true},
		{"//test.com", "http://test.com:5000", true},
		{"//test.com", "http://test.com/foobar", true},
		{"//test.com", "http://test.com/foobar/bazbar", true},
		{"//test.com", "https://test.com/foobar", true},
		{"//test.com", "https://www.test.com", false},
		{"//test.com", "www.test.com", false},

		// Wildcard host binding.
		{"http://*.test.com", "http://test.com", false}, // Should this be true?
		{"http://*.test.com", "http://notmytest.com", false},
		{"http://*.test.com", "http://www.test.com", true},
		{"http://*.test.com", "http://about.test.com", true},
		{"http://*.test.com", "http://one.two.three.test.com", true},
		{"http://www.*.test.com", "http://www.three.test.com", false},

		// Host and scheme binding (implies port).
		{"http://test.com", "http://test.com/foo", true},
		{"http://test.com", "https://test.com/foo", false},

		// Full host and port binding.
		{"http://test.com:5000", "http://test.com/foo", false},
		{"http://test.com:5000", "http://test.com:5000/foo", true},

		// Host and path binding.
		{"http://test.com/foo", "http://test.com/foo", true},
		{"http://test.com/foo", "http://test.com/foo/", true},
		{"http://test.com/foo", "http://test.com/foo/bar", true},
		{"http://test.com/foo", "http://test.com/baz", false},

		// Path only binding.
		{"/foo", "http://test.com/foo", true},
		{"/foo", "http://google.com/foo/bar", true},
		{"/foo", "http://google.com/baz/foo/bar", false},

		// Port only binding.
		{"//:5000", "http://test.com:5000/foo", true},
		{"//:5000", "https://google.com:5000/foo/bar", true},
		{"//:5000", "http://google.com/baz/foo/bar", false},

		// Port 80 is implied for HTTP.
		{"http://test.com", "http://test.com:80/foo", true},
		{"http://test.com:80", "http://test.com/foo", true},
		{"http://test.com:80", "http://test.com:80/foo", true},
		{"http://test.com:80", "http://test.com:5000/foo", false},

		// Port 443 is implied for HTTPS.
		{"https://test.com:443", "https://test.com/foo", true},
		{"https://test.com:443", "https://test.com:443/foo", true},
		{"https://test.com:443", "http://test.com/foo", false},
		{"https://test.com:443", "http://test.com:443/foo", false},
		{"https://test.com:443", "https://test.com:5000/foo", false},

		// Query param binding.
		{"?staging=true", "http://test.com/?staging=true", true},
		{"?staging=true", "http://test.com/?staging=true&debug=true", true},
		{"?staging=true", "http://test.com/?staging=false", false},
		{"?staging=true", "http://test.com/?staging=false&staging=true", false},
		{"?staging=true", "http://test.com/?staging=1", false},
		{"?staging=true", "http://test.com/", false},
		{"?lang=en&country=us", "http://test.com/?lang=en&country=us", true},
		{"?lang=en&country=us", "http://test.com/?country=us&lang=en", true},
		{"?lang=en&country=us", "http://test.com/?lang=en", false},
		{"?lang=en&country=us", "http://test.com/?country=us", false},

		// URLs without '//' are hostless, test.com is actually the path.
		{"//test.com", "test.com", false},
	}

	for _, tt := range urlTests {
		um := &urlMatcher{url: mustParseURL(tt.matchurl)}
		actual, reason := um.MatchWithReason(mustParseURL(tt.requrl))
		if actual != tt.expected {
			t.Errorf("matching '%s' against '%s' => %v, want %v (%s)", tt.requrl, tt.matchurl, actual, tt.expected, reason)
		}
	}
}
