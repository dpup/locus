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

	// VerboseLogging specifies that additional request details should be logged.
	VerboseLogging bool

	// AccessLog specifies an optional logger for request details. If nil,
	// logging goes to os.Stderr via the log package's standard logger.
	AccessLog *log.Logger

	// ErrorLog specifies an optional logger for exceptional occurances. If nil,
	// logging goes to os.Stderr via the log package's standard logger.
	ErrorLog *log.Logger

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
	rp.elogf("Starting RevProxy on port %d", port)
	return s.ListenAndServe()
}

func (rp *RevProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rp.proxy.ServeHTTP(rw, req)
}

func (rp *RevProxy) director(req *http.Request) {
	for _, c := range rp.configs {
		if c.Matches(req) {
			err := c.Transform(req)
			if err != nil { // TODO: Render local error page.
				rp.elogf("Error transforming request: %s", err)
			}
			if rp.VerboseLogging {
				d, _ := httputil.DumpRequestOut(req, false)
				rp.alogf("locus[%s] %s %s://%s %s", c.Name, req.RemoteAddr, req.URL.Scheme, req.URL.Host, string(d))
			} else {
				rp.alogf("locus[%s] %s %s %s://%s%s", c.Name, req.RemoteAddr, req.Method, req.URL.Scheme, req.URL.Host, req.URL.Path)
			}
			return
		}
	}
}

func (rp *RevProxy) alogf(format string, args ...interface{}) {
	if rp.AccessLog != nil {
		rp.AccessLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

func (rp *RevProxy) elogf(format string, args ...interface{}) {
	if rp.ErrorLog != nil {
		rp.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}
