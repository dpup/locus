package locus

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// SampleYAMLConfig demonstrates using YAML to define site configs which can be
// loaded from a file. It is also used in tests as a canonical example that
// exercises all options.
const SampleYAMLConfig = `
# The 'global' section contains settings to be applied to all sites.
global:
  add_header:
    X-Proxied-For: Locus
# The 'sites' section allows multiple configurations
sites:
  # 'about_us' is a single upstream site that sets some cookies.
  - name: about_us
    match: http://us.mysite.com/about
    upstream: http://about-1.mysite.com
    strip_header:
      - Cookie
      - User-Agent
    set_header:
      Accept-Language: en-US
  # 'search' is a site with multiple fixed upstreams.
  - name: search
    match: http://www.mysite.com/search
    upstream_set:
      - http://search-1.mysite.com
      - http://search-2.mysite.com
      - http://search-3.mysite.com
    round_robin: true
  # 'fallthrough' is a site that uses DNS to fetch multiple upstream hosts and
  # handles all other requests to mysite.com. A single upstream without a scheme
  # demarks a DNS upstream.
  - name: fallthrough
    match: http://www.mysite.com/
    upstream: dns.test.fake
    upstream_port: 4000
    upstream_path: /2016/mysite/
    ttl: 5m
    allow_stale: true
    round_robin: true
`

type yamlSiteConfig struct {
	Name         string            `yaml:"name"`
	Match        string            `yaml:"match"`
	Upstream     string            `yaml:"upstream"`
	UpstreamSet  []string          `yaml:"upstream_set"`
	RoundRobin   bool              `yaml:"round_robin"`
	UpstreamPath string            `yaml:"upstream_path"` // For DNS upstream only
	UpstreamPort uint16            `yaml:"upstream_port"` // For DNS upstream only
	TTL          time.Duration     `yaml:"ttl"`           // For DNS upstream only
	AllowStale   bool              `yaml:"allow_stale"`   // For DNS upstream only
	AddHeaders   map[string]string `yaml:"add_header"`
	SetHeaders   map[string]string `yaml:"set_header"`
	StripHeaders []string          `yaml:"strip_header"`
}

type yamlConfig struct {
	Global yamlSiteConfig   `yaml:"global"`
	Sites  []yamlSiteConfig `yaml:"sites"`
}

func loadConfigsFromYAML(data []byte) ([]*Config, error) {
	cfgs := []*Config{}

	yc := yamlConfig{}
	err := yaml.Unmarshal(data, &yc)
	if err != nil {
		return nil, fmt.Errorf("error loading YAML: %s", err)
	}

	globalCfg := &Config{}
	err = siteFromYAML(yc.Global, globalCfg)
	if err != nil {
		return nil, fmt.Errorf("error loading global cfg: %s", err)
	}

	for _, site := range yc.Sites {
		cfg := &Config{}
		*cfg = *globalCfg
		err := siteFromYAML(site, cfg)
		if err != nil {
			return nil, fmt.Errorf("error loading config: %s", err)
		}
		if cfg.upstreamProvider == nil {
			return nil, fmt.Errorf("missing upstream in %s, must specify one of 'upstream' or 'upstream_set'", cfg.Name)
		}
		cfgs = append(cfgs, cfg)
	}

	return cfgs, nil
}

func siteFromYAML(site yamlSiteConfig, cfg *Config) error {
	cfg.Name = site.Name

	err := cfg.Bind(site.Match)
	if err != nil {
		return err
	}

	up, err := upstreamFromYAML(site)
	if err != nil {
		return err
	}
	if up != nil {
		cfg.Upstream(up)
	}

	for key, value := range site.AddHeaders {
		cfg.AddHeader(key, value)
	}

	for key, value := range site.SetHeaders {
		cfg.SetHeader(key, value)
	}

	for _, key := range site.StripHeaders {
		cfg.StripHeader(key)
	}

	return nil
}

func upstreamFromYAML(site yamlSiteConfig) (UpstreamProvider, error) {
	// Because of lack of polymorphic YAML entries, there are two possible places
	// to look for upstreams. But the presence of both is invalid.
	var u UpstreamProvider
	if site.Upstream == "" && site.UpstreamSet == nil {
		return nil, nil
	} else if site.Upstream != "" && site.UpstreamSet != nil {
		return nil, errors.New("must specify one of 'upstream' or 'upstream_set' not both")
	} else if site.UpstreamSet != nil && site.RoundRobin {
		u = RoundRobin(site.UpstreamSet)
	} else if site.UpstreamSet != nil {
		u = Random(site.UpstreamSet)
	} else {
		if strings.Contains(site.Upstream, "//") {
			// Looks like full URL so treat as single upstream
			u = Single(site.Upstream)
		} else {
			// Otherwise assume upstream is a host and use it for a DNS provider.
			ds := &DNSSet{
				DNSHost:    site.Upstream,
				Port:       80,
				PathPrefix: site.UpstreamPath,
				RoundRobin: site.RoundRobin,
				AllowStale: site.AllowStale,
				TTL:        site.TTL,
			}
			if site.UpstreamPort != 0 {
				ds.Port = site.UpstreamPort
			}
			u = ds
		}
	}

	// Pre-emptively check there are no errors fetching upstreams. For fixed, this
	// is simply verifying the URLs are valid. For others it'll make a request for
	// the upstreams.
	_, err := u.All()
	if err != nil {
		return nil, fmt.Errorf("invalid upstream: %s", err)
	}

	return u, nil
}
