package locus

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"path"
	"time"
)

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

	configs []*Config
	proxy   *reverseProxy
}

// New returns an empty instance of a Locus.
func New() *Locus {
	locus := &Locus{}
	locus.TmplPath = "github.com/dpup/locus/tmpl"
	locus.configs = []*Config{}
	locus.proxy = &reverseProxy{}
	return locus
}

// NewConfig creates an empty config, registers it, then returns it.
func (locus *Locus) NewConfig() *Config {
	cfg := &Config{Name: fmt.Sprintf("cfg%d", len(locus.configs))}
	locus.AddConfig(cfg)
	return cfg
}

// AddConfig adds config to the reverse proxy. Configs will be checked in the
// order they were added, the first matching config being used to route the
// request.
func (locus *Locus) AddConfig(cfg *Config) {
	locus.configs = append(locus.configs, cfg)
}

// LoadConfigs adds site configs stored as YAML. See SampleYAMLConfig.
func (locus *Locus) LoadConfigs(data []byte) error {
	cfgs, err := loadConfigsFromYAML(data)
	if err != nil {
		return err
	}
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
	return locus.LoadConfigs(data)
}

// Serve starts a server.
func (locus *Locus) Serve(port uint16, readTimeout, writeTimeout time.Duration) error {
	s := http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        locus,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: 1 << 20,
	}
	locus.elogf("Starting Locus on port %d", port)
	return s.ListenAndServe()
}

func (locus *Locus) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
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
			err = tmpl.ExecuteTemplate(rw, "configs", locus.configs)
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
	for _, c := range locus.configs {
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
