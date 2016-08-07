package locus

import (
	"net/http"
)

// Implements and wraps a http.ResponseWriter, recording the status code.
type recordingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *recordingResponseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *recordingResponseWriter) Status() int {
	if rw.status == 0 {
		return http.StatusOK
	}
	return rw.status
}
