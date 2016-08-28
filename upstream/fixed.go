package upstream

import (
	"fmt"
	"net/url"
	"sync"
)

// FixedSet returns an upstream source that stores a fixed set of URLs, and
// caches parsed URLs and any error that occurred.
// Example use:
//     cfg.Upstream(RoundRobin(FixedSet(
//       "back-1.test.com",
//       "back-2.test.com",
//       "back-3.test.com",
//     )))
func FixedSet(urlStrs ...string) Source {
	return &fixedSet{URLStrs: urlStrs}
}

type fixedSet struct {
	URLStrs []string

	urls []*url.URL
	err  error
	mu   sync.Mutex
}

// DebugInfo returns whether the set is in an error state.
func (ru *fixedSet) DebugInfo() map[string]string {
	m := map[string]string{}
	if ru.err != nil {
		m["error"] = ru.err.Error()
	}
	return m
}

// All returns all upsteams.
func (ru *fixedSet) All() ([]*url.URL, error) {
	if ru.urls == nil && ru.err == nil {
		ru.parseURLs()
	}
	return ru.urls, ru.err
}

func (ru *fixedSet) parseURLs() {
	ru.mu.Lock()
	defer ru.mu.Unlock()
	if ru.urls != nil || ru.err != nil {
		// Calculation done in a racing thread.
		return
	}
	urls := make([]*url.URL, len(ru.URLStrs))
	for i, urlStr := range ru.URLStrs {
		u, err := url.Parse(string(urlStr))
		if err != nil {
			ru.err = fmt.Errorf("unable to parse '%s': %s", urlStr, err)
			return
		}
		urls[i] = u
	}
	ru.urls = urls
}
