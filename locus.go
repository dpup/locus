package locus

import (
	"fmt"
	"io/ioutil"
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
	cfg := &Config{Name: fmt.Sprintf("cfg%d", len(rp.configs))}
	rp.AddConfig(cfg)
	return cfg
}

// AddConfig adds config to the reverse proxy. Configs will be checked in the
// order they were added, the first matching config being used to route the
// request.
func (rp *RevProxy) AddConfig(cfg *Config) {
	rp.configs = append(rp.configs, cfg)
}

// LoadConfigs adds site configs stored as YAML. See SampleYAMLConfig.
func (rp *RevProxy) LoadConfigs(data []byte) error {
	cfgs, err := loadConfigsFromYAML(data)
	if err != nil {
		return err
	}
	for _, cfg := range cfgs {
		rp.AddConfig(cfg)
	}
	return nil
}

// LoadConfigFile reads configs from a YAML file.
func (rp *RevProxy) LoadConfigFile(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return rp.LoadConfigs(data)
}

// Serve starts a server.
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
	for _, c := range rp.configs {
		if c.Matches(req) {
			err := c.Transform(req)
			if err != nil {
				// TODO: Render local error page.
				log.Printf("Error transforming request: %s", err)
			}
			if rp.VerboseLogging {
				d, _ := httputil.DumpRequestOut(req, false)
				log.Printf("config[%s] %s://%s %s", c.Name, req.URL.Scheme, req.URL.Host, string(d))
			} else {
				log.Printf("config[%s] %s %s://%s%s", c.Name, req.Method, req.URL.Scheme, req.URL.Host, req.URL.Path)
			}
			return
		}
	}
}
