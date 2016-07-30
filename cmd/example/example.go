package main

import (
	"github.com/dpup/locus"
	"time"
)

func main() {

	proxy := locus.New()
	proxy.VerboseLogging = true

	// open http://localhost:5555/news/world-middle-east-36932694
	bbc := proxy.NewConfig()
	bbc.Match("/news")
	bbc.Upstream("http://www.bbc.com/news")
	bbc.StripHeader("Cookie")
	bbc.SetHeader("Host", "www.bbc.com")
	bbc.SetHeader("Referer", "http://www.bbc.com/news")

	// open localhost:5555/wiki/England
	wiki := proxy.NewConfig()
	wiki.Match("/wiki")
	wiki.Upstream("https://en.wikipedia.org")
	wiki.StripHeader("Cookie")
	wiki.SetHeader("Host", "en.wikipedia.org")
	wiki.SetHeader("Referer", "https://en.wikipedia.org/wiki/Main_Page")

	panic(proxy.Serve(5555, 10*time.Second, 10*time.Second))
}
