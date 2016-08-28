package upstream

import (
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
