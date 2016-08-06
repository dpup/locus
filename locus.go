package locus

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"time"
)

// Query param that when specified in the URL overrides the request in the URL.
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

	// TmplPath specifies the location of HTML templates.
	TmplPath string

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
		TmplPath:     "github.com/dpup/locus/tmpl", // TODO: compile in?
		Port:         5555,
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
	}
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

// LoadConfig adds site configs stored as YAML. See SampleYAMLConfig.
func (locus *Locus) LoadConfig(data []byte) error {
	cfgs, globals, err := loadConfigFromYAML(data)
	if err != nil {
		return err
	}
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
	for _, cfg := range cfgs {
		locus.AddConfig(cfg)
	}
	return nil
}

// LoadConfigFile reads configs from a YAML file.
func (locus *Locus) LoadConfigFile(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return locus.LoadConfig(data)
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
			return
		}

		if err := locus.proxy.Proxy(rw, proxyreq); err != nil { // TODO: Render local error page.
			locus.elogf("error proxying request: %v", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		if locus.VerboseLogging {
			d, _ := httputil.DumpRequestOut(proxyreq, false)
			locus.alogf("locus[%s] %s %s://%s %s", c.Name, proxyreq.RemoteAddr, proxyreq.URL.Scheme, proxyreq.URL.Host, string(d))
		} else {
			locus.alogf("locus[%s] %s %s %s://%s%s", c.Name, proxyreq.RemoteAddr, proxyreq.Method, proxyreq.URL.Scheme, proxyreq.URL.Host, proxyreq.URL.Path)
		}
	} else if req.URL.Path == "/debug/configs" {
		// Renders a debug page with information about the configs.
		tmpl, err := locus.loadTemplates()
		if err == nil {
			err = tmpl.ExecuteTemplate(rw, "configs", locus)
		}
		if err != nil { // TODO: Render local error page.
			locus.elogf("error rendering upstreams debug page: %v", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		rw.WriteHeader(http.StatusNotImplemented)
	}
}

func (locus *Locus) findConfig(req *http.Request) *Config {
	for _, c := range locus.Configs {
		if c.Matches(req) {
			return c
		}
	}
	return nil
}

func (locus *Locus) loadTemplates() (*template.Template, error) {
	// TODO: cache.
	return template.New("root").ParseGlob(path.Join(locus.TmplPath, "*.html"))
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
