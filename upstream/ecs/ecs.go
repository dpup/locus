// Package ecs registers an upstream that uses an ECS service. Using the AWS SDK
// it will poll for changes to running tasks, and allow traffic to be load
// balanced across multiple tasks, without needing an ELB.
//
// If you want to use an ECS upstream, register by importing this package:
//
//        import _ "github.com/dpup/locus/upstream/ecs"
//
// And then specify the upstream location in the form:
// ecs://service.cluster.us-east-1
package ecs

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/dpup/esu"
	"github.com/dpup/locus/upstream"
)

func init() {
	upstream.Register(
		`^ecs://[\w\.-]+\.[\w\.-]+\.[\w\.-]+$`,
		func(location string, settings map[string]string) (upstream.Source, error) {
			u, err := url.Parse(location)
			if err != nil {
				return nil, fmt.Errorf("failed to parse URL %s: %s", location, err)
			}
			if u.Scheme != "ecs" {
				return nil, fmt.Errorf("expect ecs:// scheme, was %s", location)
			}
			parts := strings.Split(u.Host, ".")
			if len(parts) != 3 {
				return nil, fmt.Errorf("host should be service.cluster.region, was %s", location)
			}

			service, cluster, region := parts[0], parts[1], parts[2]

			sess, err := getSession(region)
			if err != nil {
				return nil, fmt.Errorf("unable to create session for %s: %s", location, err)
			}

			var allowStale bool

			if a, ok := settings["allow_stale"]; ok {
				ai, err := strconv.ParseBool(a)
				if err != nil {
					return nil, fmt.Errorf("invalid boolean for allow_stale '%s', %s", a, err)
				}
				allowStale = ai
			}

			e := &ECS{
				location:   u,
				allowStale: allowStale,
				path:       settings["path"],
				// TODO: AllowStale
			}

			monitor := esu.NewTaskMonitor(sess, cluster, service)
			monitor.OnTaskChange = e.onTaskChange
			monitor.OnError = e.onMonitorError
			monitor.Monitor()

			return e, nil
		})
}

// ECS is a....
type ECS struct {
	location   *url.URL
	allowStale bool
	path       string
	log        *log.Logger
	tasks      []esu.TaskInfo
	urls       []*url.URL
	updateAt   time.Time
	err        error
	errAt      time.Time
}

// DebugInfo returns extra fields to show on /debug/configs
func (e *ECS) DebugInfo() map[string]string {
	m := map[string]string{}
	m["location"] = e.location.String()
	m["allow stale"] = fmt.Sprintf("%v", e.allowStale)
	m["last update"] = e.updateAt.Format(time.Stamp)
	if e.err != nil {
		// TODO: Figure out a better way of dealing with error cases.
		m["error"] = e.err.Error()
		m["error at"] = e.errAt.Format(time.Stamp)
	}
	for i, t := range e.tasks {
		m[fmt.Sprintf("ecs task #%d", i)] = t.String()
	}
	return m
}

// All returns all upsteams.
func (e *ECS) All() ([]*url.URL, error) {
	if e.err != nil && !e.allowStale {
		return nil, e.err
	}
	return e.urls, nil
}

func (e *ECS) onTaskChange(tasks []esu.TaskInfo) {
	e.tasks = tasks
	e.updateAt = time.Now()
	e.err = nil

	e.urls = make([]*url.URL, len(tasks))
	for i, t := range tasks {
		e.urls[i] = &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", t.PublicIPAddress, t.Port),
			Path:   e.location.Path,
		}
	}
}

func (e *ECS) logf(format string, args ...interface{}) {
	if e.log != nil {
		e.log.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

func (e *ECS) onMonitorError(err error) {
	e.err = err
	e.errAt = time.Now()
	e.logf("monitor error for %s: %v", e.location, err)
}

var sessions = map[string]*session.Session{}

// getSession returns an AWS session using information from the environment. One
// session is created per region and cached globally. It is assumed this is
// called during setup and as such shouldn't be accessed concurrently.
func getSession(region string) (*session.Session, error) {
	if sess, ok := sessions[region]; ok {
		return sess, nil
	}
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
		CredentialsChainVerboseErrors: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	sessions[region] = sess
	return sess, nil
}
