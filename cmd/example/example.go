package main

import (
	"github.com/dpup/locus"
)

func main() {

	proxy := locus.New()

	wiki := proxy.NewConfig()
	wiki.Match("/wiki")
	wiki.Upstream("https://en.wikipedia.org/wiki/")
	wiki.StripHeader("Cookie")
	wiki.SetHeader("Referer", "https://en.wikipedia.org/wiki/Main_Page")
}
