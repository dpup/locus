package locus

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

// RevProxy wraps golang's httputil.ReverseProxy to provide multi-host routing.
type RevProxy struct {
	VerboseLogging bool
	configs        []*Config
	proxy          *httputil.ReverseProxy
}

// New returns an empty instance of a RevProxy.
func New() *RevProxy {
	rp := &RevProxy{}
	rp.configs = []*Config{}
	rp.proxy = &httputil.ReverseProxy{Director: rp.director}
	return rp
}

// NewConfig creates an empty config, registers it, then returns it.
func (rp *RevProxy) NewConfig() *Config {
	cfg := &Config{}
	rp.AddConfig(cfg)
	return cfg
}

// AddConfig adds config to the reverse proxy. Configs will be checked in the
// order they were added, the first matching config being used to route the
// request.
func (rp *RevProxy) AddConfig(cfg *Config) {
	rp.configs = append(rp.configs, cfg)
}

// Serve starts a server on
func (rp *RevProxy) Serve(port uint16, readTimeout, writeTimeout time.Duration) error {
	s := http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        rp,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: 1 << 20,
	}
	log.Printf("Starting RevProxy on port %d", port)
	return s.ListenAndServe()
}

func (rp *RevProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rp.proxy.ServeHTTP(rw, req)
}

func (rp *RevProxy) director(req *http.Request) {
	for i, c := range rp.configs {
		if c.RequestMatcher.Matches(*req) {
			c.Transform(req)
			if rp.VerboseLogging {
				d, _ := httputil.DumpRequestOut(req, false)
				log.Printf("Config(%d) %s", i, string(d))
			} else {
				log.Printf("Config(%d) %s %s://%s%s", i, req.Method, req.URL.Scheme, req.URL.Host, req.URL.Path)
			}
			return
		}
	}
}
