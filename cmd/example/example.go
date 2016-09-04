package main

import (
	"github.com/dpup/locus"
	"github.com/dpup/locus/upstream"
	"net/http"
)

func main() {

	proxy := locus.New()
	proxy.VerboseLogging = true

	// open http://localhost:5555/news/world-middle-east-36932694
	bbc := proxy.NewConfig()
	bbc.Bind("/news")
	bbc.Upstream(upstream.Single("http://www.bbc.com/news"))
	bbc.StripHeader("Cookie") // Avoid fowarding localhost cookies.
	bbc.SetHeader("Host", "www.bbc.com")
	bbc.SetHeader("Referer", "http://www.bbc.com/news")

	// open localhost:5555/wiki/England
	wiki := proxy.NewConfig()
	wiki.Bind("/wiki")
	wiki.Upstream(upstream.Random(upstream.FixedSet(
		"https://en.wikipedia.org",
		"https://www.wikipedia.org",
	)))
	wiki.StripHeader("Cookie")
	wiki.SetHeader("Host", "en.wikipedia.org")
	wiki.SetHeader("Referer", "https://en.wikipedia.org/wiki/Main_Page")

	// open localhost:5555/amazon/dogs
	amazon := proxy.NewConfig()
	amazon.Bind("/amazon")
	amazon.Upstream(upstream.RoundRobin(&upstream.DNS{Host: "amazon.com", Path: "/404"}))
	amazon.StripHeader("Cookie")
	amazon.SetHeader("Host", "www.amazon.com")

	// open http://localhost:5555/goog/search?q=wunderbar
	goog := proxy.NewConfig()
	goog.Bind("/goog")
	goog.Upstream(upstream.Single("http://www.google.com/"))
	goog.Redirect = http.StatusFound

	panic(proxy.ListenAndServe())
}
