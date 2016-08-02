// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Forked copy of golang's standard library's HTTP reverse proxy handler

package locus

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// onExitFlushLoop is a callback set by tests to detect the state of the
// flushLoop() goroutine.
var onExitFlushLoop func()

// reverseProxy is an HTTP Handler that takes an incoming request and
// sends it to another server, proxying the response back to the
// client.
type reverseProxy struct {
	// The transport used to perform proxy requests.
	// If nil, http.DefaultTransport is used.
	Transport http.RoundTripper

	// FlushInterval specifies the flush interval
	// to flush to the client while copying the
	// response body.
	// If zero, no periodic flushing is done.
	FlushInterval time.Duration

	// BufferPool optionally specifies a buffer pool to
	// get byte slices for use by io.CopyBuffer when
	// copying HTTP response bodies.
	BufferPool BufferPool
}

// A BufferPool is an interface for getting and returning temporary
// byte slices for use by io.CopyBuffer.
type BufferPool interface {
	Get() []byte
	Put([]byte)
}

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; http://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

type requestCanceler interface {
	CancelRequest(*http.Request)
}

type runOnFirstRead struct {
	io.Reader // optional; nil means empty body

	fn func() // Run before first Read, then set to nil
}

func (c *runOnFirstRead) Read(bs []byte) (int, error) {
	if c.fn != nil {
		c.fn()
		c.fn = nil
	}
	if c.Reader == nil {
		return 0, io.EOF
	}
	return c.Reader.Read(bs)
}

// Now only intended for use in tests.
func (p *reverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	err := p.Proxy(rw, req)
	if err != nil {
		log.Printf("proxy error:", err)
		rw.WriteHeader(http.StatusInternalServerError)
	}
}

// Proxy makes a request and proxies the response to the response writer.
func (p *reverseProxy) Proxy(rw http.ResponseWriter, proxyreq *http.Request) error {
	transport := p.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	if closeNotifier, ok := rw.(http.CloseNotifier); ok {
		if requestCanceler, ok := transport.(requestCanceler); ok {
			reqDone := make(chan struct{})
			defer close(reqDone)

			clientGone := closeNotifier.CloseNotify()

			proxyreq.Body = struct {
				io.Reader
				io.Closer
			}{
				Reader: &runOnFirstRead{
					Reader: proxyreq.Body,
					fn: func() {
						go func() {
							select {
							case <-clientGone:
								requestCanceler.CancelRequest(proxyreq)
							case <-reqDone:
							}
						}()
					},
				},
				Closer: proxyreq.Body,
			}
		}
	}

	proxyreq.Proto = "HTTP/1.1"
	proxyreq.ProtoMajor = 1
	proxyreq.ProtoMinor = 1
	proxyreq.Close = false

	// Remove hop-by-hop headers to the backend.  Especially
	// important is "Connection" because we want a persistent
	// connection, regardless of what the client sent to us.
	for _, h := range hopHeaders {
		if proxyreq.Header.Get(h) != "" {
			proxyreq.Header.Del(h)
		}
	}

	if clientIP, _, err := net.SplitHostPort(proxyreq.RemoteAddr); err == nil {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		if prior, ok := proxyreq.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		proxyreq.Header.Set("X-Forwarded-For", clientIP)
	}

	res, err := transport.RoundTrip(proxyreq)
	if err != nil {
		return fmt.Errorf("proxy error: %v", err)
	}

	for _, h := range hopHeaders {
		res.Header.Del(h)
	}

	copyHeader(rw.Header(), res.Header)

	// The "Trailer" header isn't included in the Transport's response,
	// at least for *http.Transport. Build it up from Trailer.
	if len(res.Trailer) > 0 {
		var trailerKeys []string
		for k := range res.Trailer {
			trailerKeys = append(trailerKeys, k)
		}
		rw.Header().Add("Trailer", strings.Join(trailerKeys, ", "))
	}

	rw.WriteHeader(res.StatusCode)
	if len(res.Trailer) > 0 {
		// Force chunking if we saw a response trailer.
		// This prevents net/http from calculating the length for short
		// bodies and adding a Content-Length.
		if fl, ok := rw.(http.Flusher); ok {
			fl.Flush()
		}
	}
	p.copyResponse(rw, res.Body)
	res.Body.Close() // close now, instead of defer, to populate res.Trailer
	copyHeader(rw.Header(), res.Trailer)
	return nil
}

func (p *reverseProxy) copyResponse(dst io.Writer, src io.Reader) {
	if p.FlushInterval != 0 {
		if wf, ok := dst.(writeFlusher); ok {
			mlw := &maxLatencyWriter{
				dst:     wf,
				latency: p.FlushInterval,
				done:    make(chan bool),
			}
			go mlw.flushLoop()
			defer mlw.stop()
			dst = mlw
		}
	}

	var buf []byte
	if p.BufferPool != nil {
		buf = p.BufferPool.Get()
	}
	io.CopyBuffer(dst, src, buf)
	if p.BufferPool != nil {
		p.BufferPool.Put(buf)
	}
}

type writeFlusher interface {
	io.Writer
	http.Flusher
}

type maxLatencyWriter struct {
	dst     writeFlusher
	latency time.Duration

	lk   sync.Mutex // protects Write + Flush
	done chan bool
}

func (m *maxLatencyWriter) Write(p []byte) (int, error) {
	m.lk.Lock()
	defer m.lk.Unlock()
	return m.dst.Write(p)
}

func (m *maxLatencyWriter) flushLoop() {
	t := time.NewTicker(m.latency)
	defer t.Stop()
	for {
		select {
		case <-m.done:
			if onExitFlushLoop != nil {
				onExitFlushLoop()
			}
			return
		case <-t.C:
			m.lk.Lock()
			m.dst.Flush()
			m.lk.Unlock()
		}
	}
}

func (m *maxLatencyWriter) stop() { m.done <- true }
