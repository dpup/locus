package upstream

import (
	"fmt"
	"regexp"
)

// FactoryFn will take an upstream configuration and return an upsteam Source.
type FactoryFn func(string, map[string]string) (Source, error)

type factory struct {
	pattern string
	fn      FactoryFn
}

var factories = []factory{}

// Register adds an upstream factory to the global registry.
func Register(pattern string, fn FactoryFn) {
	factories = append(factories, factory{pattern: pattern, fn: fn})
}

// Get returns an upstream Source for a given location string.
func Get(location string, settings map[string]string) (Source, error) {
	for _, f := range factories {
		ok, err := regexp.Match(f.pattern, []byte(location))
		if ok {
			return f.fn(location, settings)
		} else if err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("no upstream factory matches %s", location)
}
