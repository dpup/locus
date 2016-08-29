package upstream

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/dpup/esu"
)

// ECS is a...
// loc: ecs://service.cluster.us-east-1/foo/bar/baz
func ECS(loc string) (*ECSSet, error) {
	u, err := url.Parse(loc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %s", err)
	}
	if u.Scheme != "ecs" {
		return nil, fmt.Errorf("expect ecs:// scheme, was %s", loc)
	}
	parts := strings.Split(u.Host, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("host should be service.cluster.region, was %s", loc)
	}
	service, cluster, region := parts[0], parts[1], parts[2]

	sess, err := getSession(region)
	if err != nil {
		return nil, fmt.Errorf("unable to create session: %s", err)
	}

	e := &ECSSet{location: u}

	monitor := esu.NewTaskMonitor(sess, cluster, service)
	monitor.OnTaskChange = e.onTaskChange
	monitor.OnError = e.onMonitorError
	monitor.Monitor()

	return e, nil
}

// ECSSet is a....
type ECSSet struct {
	location *url.URL
	tasks    []esu.TaskInfo
	urls     []*url.URL
	updateAt time.Time
	err      error
	errAt    time.Time
}

// DebugInfo returns extra fields to show on /debug/configs
func (e *ECSSet) DebugInfo() map[string]string {
	m := map[string]string{}
	m["location"] = e.location.String()
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
func (e *ECSSet) All() ([]*url.URL, error) {
	return e.urls, e.err
}

func (e *ECSSet) onTaskChange(tasks []esu.TaskInfo) {
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

func (e *ECSSet) onMonitorError(err error) {
	e.err = err
	e.errAt = time.Now()
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
