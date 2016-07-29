package locus

import (
	"net/http"
	"net/url"
	"testing"
)

func TestUrlMatcher(t *testing.T) {
	var urlTests = []struct {
		matchurl string
		requrl   string
		expected bool
	}{
		{"//test.com", "test.com", false},

		{"//test.com", "//test.com", true},
		{"//test.com", "http://test.com", true},
		{"//test.com", "https://test.com", true},
		{"//test.com", "ftp://test.com", true},
		{"//test.com", "http://test.com/foobar", true},
		{"//test.com", "http://test.com/foobar/bazbar", true},
		{"//test.com", "https://test.com/foobar", true},
		{"//test.com", "https://www.test.com", false},
		{"//test.com", "www.test.com", false},

		{"http://test.com", "http://test.com/foo", true},
		{"http://test.com", "https://test.com/foo", false},

		{"http://test.com/foo", "http://test.com/foo", true},
		{"http://test.com/foo", "http://test.com/foo/", true},
		{"http://test.com/foo", "http://test.com/foo/bar", true},
		{"http://test.com/foo", "http://test.com/baz", false},

		{"http://test.com:5000", "http://test.com/foo", false},
		{"http://test.com:5000", "http://test.com:5000/foo", true},

		// Normalize port 80 to make it optional.
		{"http://test.com", "http://test.com:80/foo", true},
		{"http://test.com:80", "http://test.com/foo", true},
		{"http://test.com:80", "http://test.com:80/foo", true},
		{"http://test.com:80", "http://test.com:5000/foo", false},

		// Make 443 optional for https.
		{"https://test.com:443", "https://test.com/foo", true},
		{"https://test.com:443", "https://test.com:443/foo", true},
		{"https://test.com:443", "http://test.com/foo", false},
		{"https://test.com:443", "http://test.com:443/foo", false},
		{"https://test.com:443", "https://test.com:5000/foo", false},
	}

	for _, tt := range urlTests {
		um := &urlMatcher{url: mustParseURL(tt.matchurl)}
		actual, reason := um.MatchWithReason(mustReq(tt.requrl))
		if actual != tt.expected {
			t.Errorf("matching '%s' against '%s' => %v, want %v (%s)", tt.requrl, tt.matchurl, actual, tt.expected, reason)
		}
	}
}

func mustParseURL(rawurl string) url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return *u
}

func mustReq(rawurl string) http.Request {
	r, err := http.NewRequest("GET", rawurl, nil)
	if err != nil {
		panic(err)
	}
	return *r
}
