package main

import (
	"flag"
	"log"
	"os"

	"github.com/dpup/locus"
)

var conf = flag.String("conf", "/etc/locus.conf", "Location of config file.")

func main() {
	flag.Parse()

	proxy, err := locus.FromConfigFile(*conf)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	if err := proxy.ListenAndServe(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}