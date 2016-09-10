package main

import (
	_ "expvar"
	"flag"
	"log"
	"os"

	"github.com/dpup/locus"
	_ "github.com/dpup/locus/upstream/ecs"
)

var conf = flag.String("conf", "/etc/locus.conf", "Location of config file.")

func main() {
	flag.Parse()

	proxy, err := locus.FromConfigFile(*conf)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	proxy.RegisterMetricsWithDefaultRegistry()
	if err := proxy.ListenAndServe(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
