package locus

import (
	"net/http"
	"net/http/httputil"
)

// RevProxy wraps golang's httputil.ReverseProxy to provide multi-host routing.
type RevProxy struct {
	configs []*Config
	proxy   *httputil.ReverseProxy
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

func (rp *RevProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rp.proxy.ServeHTTP(rw, req)
}

func (rp *RevProxy) director(req *http.Request) {
	for _, c := range rp.configs {
		if c.RequestMatcher.Matches(*req) {
			c.Transform(req)
			return
		}
	}
}
