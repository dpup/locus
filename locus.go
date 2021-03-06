// Package locus provides a multi-host reverse proxy.
package locus

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/dpup/locus/tmpl"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
)

// HostOverrideParam param that when specified in the querystring overrides the
// host in the requested URL. Intended for testing staged sites.
// e.g. http://localhost:5555/?locus_host=sample.locus.xyz
const HostOverrideParam = "locus_host"

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

	Requests    metrics.Meter
	Errors      metrics.Meter
	Connections metrics.Counter
	Latency     metrics.Histogram

	proxy *reverseProxy
}

// New returns an instance of a Locus server with the following defaults set:
// Port = 5555
// ReadTimeout = 30s
// WriteTimeout = 30s
func New() *Locus {
	locus := &Locus{
		Configs:      []*Config{},
		Port:         5555,
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,

		proxy:       &reverseProxy{},
		Requests:    metrics.NewMeter(),
		Errors:      metrics.NewMeter(),
		Connections: metrics.NewCounter(),
		Latency:     metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015)),
	}
	return locus
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
		ErrorLog:       locus.ErrorLog,
	}
	locus.elogf("Starting Locus on port %d", locus.Port)
	return s.ListenAndServe()
}

func (locus *Locus) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	locus.maybeApplyHostOverride(req)

	locus.Requests.Mark(1)
	locus.Connections.Inc(1)
	now := time.Now()
	defer func() {
		locus.Connections.Dec(1)
		locus.Latency.Update(int64(time.Since(now) / time.Millisecond))
	}()

	rrw := &recordingResponseWriter{ResponseWriter: rw}

	c := locus.findConfig(req)
	if c != nil {
		// Found matching config so get a request for proxying.
		proxyreq, err := c.Direct(req)

		if err != nil {
			locus.elogf("error transforming request: %v", err)
			locus.renderError(rrw, http.StatusInternalServerError)
			locus.logDefaultReq(rrw, req)
			return
		}

		var d string

		if c.Redirect != 0 {
			rrw.Header().Add("Location", proxyreq.URL.String())
			rrw.WriteHeader(c.Redirect)

		} else {
			if err := locus.proxy.Proxy(rrw, proxyreq); err != nil { // TODO: Render local error page.
				locus.elogf("error proxying request: %v", err)
				locus.renderError(rrw, http.StatusBadGateway)
			}
			d = locus.maybeDumpRequest(req)
		}

		locus.alogf("locus[%s] %d %s %s %s => %s - %s %q %s",
			c.Name, rrw.Status(), req.Method, req.Host, req.URL, proxyreq.URL, remoteAddr(req),
			req.Header.Get("User-Agent"), string(d))

	} else if req.URL.Path == "/debug/configs" {
		tmpl.ConfigsTemplate.Execute(rw, locus)
		locus.logDefaultReq(rrw, req)

	} else if req.URL.Path == "/debug/vars" {
		// In Go1.8 add expvars handler directly. See https://github.com/golang/go/issues/15030
		http.DefaultServeMux.ServeHTTP(rw, req)

		// For legacy healthchecking, render 200 on root path.
	} else if req.URL.Path == "/" {
		locus.renderError(rrw, http.StatusOK)
		locus.logDefaultReq(rrw, req)

	} else {
		locus.renderError(rrw, http.StatusNotFound)
		locus.logDefaultReq(rrw, req)
	}
}

// RegisterMetrics adds locus metrics to the metrics registry.
func (locus *Locus) RegisterMetrics(m metrics.Registry) {
	m.Register("requests", locus.Requests)
	m.Register("errors", locus.Errors)
	m.Register("conns", locus.Connections)
	m.Register("latency", locus.Latency)

	exp.Exp(m)
	go metrics.Log(m, 60*time.Second, locus.ErrorLog)
}

// RegisterMetricsWithDefaultRegistry registers metrics with the default registry.
func (locus *Locus) RegisterMetricsWithDefaultRegistry() {
	locus.RegisterMetrics(metrics.DefaultRegistry)
}

func (locus *Locus) maybeApplyHostOverride(req *http.Request) {
	q := req.URL.Query()
	overrideParam := q.Get(HostOverrideParam)
	if overrideParam != "" {
		req.Host = overrideParam

		// Avoid infinite loops.
		q.Del(HostOverrideParam)
		req.URL.RawQuery = q.Encode()
	}
}

func (locus *Locus) findConfig(req *http.Request) *Config {
	for _, c := range locus.Configs {
		if ok, _ := c.Match(req); ok {
			return c
		}
	}
	return nil
}

func (locus *Locus) renderError(rw http.ResponseWriter, status int) {
	if status >= 500 {
		locus.Errors.Mark(1)
	}
	rw.WriteHeader(status)
	tmpl.ErrorTemplate.Execute(rw, struct {
		Status int
	}{status})
}

func (locus *Locus) logDefaultReq(rw *recordingResponseWriter, req *http.Request) {
	locus.alogf("locus[-] %d %s %s %s - %s %q %s", rw.Status(), req.Method, req.Host, req.URL,
		remoteAddr(req), req.Header.Get("User-Agent"), locus.maybeDumpRequest(req))
}

func (locus *Locus) maybeDumpRequest(req *http.Request) string {
	if locus.VerboseLogging {
		d, err := httputil.DumpRequest(req, false)
		if err == nil {
			return string(d)
		}
		locus.elogf("failed to dump request: %v", err)
	}
	return ""
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
