package locus

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/dpup/locus/tmpl"
)

// OverrideQueryParam param that when specified in the URL overrides the request
// in the URL.
// e.g. http://localhost:5555/?locus_override=http://sample.locus.xyz
const OverrideQueryParam = "locus_override"

// Locus wraps a fork of golang's httputil.ReverseProxy to provide multi-host
// routing.
type Locus struct {

	// VerboseLogging specifies that additional request details should be logged.
	VerboseLogging bool

	// AccessLog specifies an optional logger for request details. If nil,
	// logging goes to os.Stderr via the log package's standard logger.
	AccessLog *log.Logger

	// ErrorLog specifies an optional logger for exceptional occurances. If nil,
	// logging goes to os.Stderr via the log package's standard logger.
	ErrorLog *log.Logger

	// Port specifies the port for incoming connections.
	Port uint16

	// ReadTimeout is the maximum duration before timing out read of the request.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out write of the
	// response.
	WriteTimeout time.Duration

	// Configs is a list of sites that locus will forward for.
	Configs []*Config

	proxy *reverseProxy
}

// New returns an instance of a Locus server with the following defaults set:
// Port = 5555
// ReadTimeout = 30s
// WriteTimeout = 30s
func New() *Locus {
	return &Locus{
		proxy:        &reverseProxy{},
		Configs:      []*Config{},
		Port:         5555,
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
	}
}

// FromConfig creates a new locus server from YAML config.
// See SampleYAMLConfig.
func FromConfig(data []byte) (*Locus, error) {
	cfgs, globals, err := loadConfigFromYAML(data)
	if err != nil {
		return nil, err
	}

	locus := New()

	if globals.Port != 0 {
		locus.Port = globals.Port
	}
	if globals.ReadTimeout != 0 {
		locus.ReadTimeout = globals.ReadTimeout
	}
	if globals.WriteTimeout != 0 {
		locus.WriteTimeout = globals.WriteTimeout
	}

	locus.VerboseLogging = globals.VerboseLogging

	if globals.AccessLog != "" {
		locus.AccessLog, err = newLogger(globals.AccessLog)
		if err != nil {
			return nil, err
		}
	}

	if globals.ErrorLog != "" {
		locus.ErrorLog, err = newLogger(globals.ErrorLog)
		if err != nil {
			return nil, err
		}
	}

	for _, cfg := range cfgs {
		locus.AddConfig(cfg)
	}

	return locus, nil
}

// FromConfigFile creates a new locus server from a YAML config file.
func FromConfigFile(filename string) (*Locus, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return FromConfig(data)
}

// NewConfig creates an empty config, registers it, then returns it.
func (locus *Locus) NewConfig() *Config {
	cfg := &Config{Name: fmt.Sprintf("cfg%d", len(locus.Configs))}
	locus.AddConfig(cfg)
	return cfg
}

// AddConfig adds config to the reverse proxy. Configs will be checked in the
// order they were added, the first matching config being used to route the
// request.
func (locus *Locus) AddConfig(cfg *Config) {
	locus.Configs = append(locus.Configs, cfg)
}

// ListenAndServe listens on locus.Port for incoming connections.
func (locus *Locus) ListenAndServe() error {
	s := http.Server{
		Addr:           fmt.Sprintf(":%d", locus.Port),
		Handler:        locus,
		ReadTimeout:    locus.ReadTimeout,
		WriteTimeout:   locus.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}
	locus.elogf("Starting Locus on port %d", locus.Port)
	return s.ListenAndServe()
}

func (locus *Locus) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	overrideParam := req.URL.Query().Get(OverrideQueryParam)
	if overrideParam != "" {
		overrideURL, err := url.Parse(overrideParam)
		if err != nil {
			locus.elogf("error parsing override URL, ignoring: ", err)
		} else {
			req.URL = overrideURL
		}
	}

	c := locus.findConfig(req)
	if c != nil {
		// Found matching config so copy req, transform it, and forward it.
		proxyreq := copyRequest(req)

		if err := c.Transform(proxyreq); err != nil { // TODO: Render local error page.
			locus.elogf("error transforming request: %v", err)
			rw.WriteHeader(http.StatusInternalServerError)
			locus.logDefaultReq(http.StatusInternalServerError, req)
			return
		}

		status := http.StatusOK // TODO: extract status code from rw.

		if err := locus.proxy.Proxy(rw, proxyreq); err != nil { // TODO: Render local error page.
			locus.elogf("error proxying request: %v", err)
			rw.WriteHeader(http.StatusInternalServerError)
			status = http.StatusInternalServerError
		}

		var d []byte
		if locus.VerboseLogging {
			d, _ = httputil.DumpRequestOut(proxyreq, false)
		}
		locus.alogf("locus[%s] %d %s %s => %s (%s \"%s\") %s",
			c.Name, status, req.Method, req.URL, proxyreq.URL, req.RemoteAddr,
			req.Header.Get("User-Agent"), string(d))

	} else if req.URL.Path == "/debug/configs" {
		tmpl.DebugTemplate.ExecuteTemplate(rw, "configs", locus)
		locus.logDefaultReq(http.StatusOK, req)
	} else {
		rw.WriteHeader(http.StatusNotImplemented)
		locus.logDefaultReq(http.StatusNotImplemented, req)
	}
}

func (locus *Locus) logDefaultReq(status int, req *http.Request) {
	locus.alogf("locus[-] %d %s %s (%s \"%s\")",
		status, req.Method, req.URL, req.RemoteAddr, req.Header.Get("User-Agent"))
}

func (locus *Locus) findConfig(req *http.Request) *Config {
	for _, c := range locus.Configs {
		if c.Matches(req) {
			return c
		}
	}
	return nil
}

func (locus *Locus) alogf(format string, args ...interface{}) {
	if locus.AccessLog != nil {
		locus.AccessLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

func (locus *Locus) elogf(format string, args ...interface{}) {
	if locus.ErrorLog != nil {
		locus.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}
