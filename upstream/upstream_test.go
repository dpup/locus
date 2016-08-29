package upstream

import (
	"net/http"
	"testing"
)

func TestRoundRobin(t *testing.T) {
	provider := RoundRobin(FixedSet(
		"back-1.test.com",
		"back-2.test.com",
		"back-3.test.com",
	))

	if u, _ := provider.Get(nil); u.String() != "back-1.test.com" {
		t.Error("First call to get should return 'back-1'")
	}
	if u, _ := provider.Get(nil); u.String() != "back-2.test.com" {
		t.Error("Second call to get should return 'back-2'")
	}
	if u, _ := provider.Get(nil); u.String() != "back-3.test.com" {
		t.Error("Third call to get should return 'back-3'")
	}
	if u, _ := provider.Get(nil); u.String() != "back-1.test.com" {
		t.Error("Forth call to get should return 'back-1'")
	}
}

func TestIPHash(t *testing.T) {
	provider := IPHash(FixedSet(
		"back-1.test.com",
		"back-2.test.com",
		"back-3.test.com",
	))

	req1, _ := http.NewRequest("GET", "xxx", nil)
	req1.RemoteAddr = "10.0.0.12"

	expected1 := "back-1.test.com"
	for i := 0; i < 10; i++ {
		if u, _ := provider.Get(req1); u.String() != expected1 {
			t.Fatalf("Unexpected backend, expected %s but was %s", expected1, u)
		}
	}

	req2, _ := http.NewRequest("GET", "xxx", nil)
	req2.RemoteAddr = "10.0.0.16"
	expected2 := "back-2.test.com"
	for i := 0; i < 10; i++ {
		if u, _ := provider.Get(req2); u.String() != expected2 {
			t.Fatalf("Unexpected backend, expected %s but was %s", expected2, u)
		}
	}

	// X-Forwarded-For takes precedence over RemoteAddr.
	req3, _ := http.NewRequest("GET", "xxx", nil)
	req3.RemoteAddr = "10.0.0.16"
	req3.Header.Set("X-Forwarded-For", "10.0.0.12")
	expected3 := "back-1.test.com"
	for i := 0; i < 10; i++ {
		if u, _ := provider.Get(req3); u.String() != expected3 {
			t.Fatalf("Unexpected backend, expected %s but was %s", expected3, u)
		}
	}
}
