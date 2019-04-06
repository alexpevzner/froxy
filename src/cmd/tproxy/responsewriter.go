//
// http.ResponseWriter wrapper with various hooks
//

package main

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

//
// http.ResponseWriter wrapper
//
type ResponseWriter struct {
	http.ResponseWriter // Underlying http.ResponseWriter

	//
	// This hook is called when HTTP error occurs
	// (i.e., returned HTTP status is 4xx or 5xx)
	//
	// If this function returns non-nil data,
	// the response body will be replaced by
	// this data
	//
	// This function is called just before response
	// headers are written, so it may modify the
	// response headers as needed
	//
	OnError func(w http.ResponseWriter, status int) []byte

	//
	// This hook is called when HTTP request was completed
	// with successful status (2xx), just before header was
	// written. It can add additional headers
	//
	// Note, this hook MUST NOT call the http.ResponseWriter.Write()
	// method
	//
	OnSuccess func(w http.ResponseWriter, status int)

	wroteHeader bool // Hider was written
	skipBody    bool // Skip the body
}

var _ = http.ResponseWriter(&ResponseWriter{})
var _ = http.Hijacker(&ResponseWriter{})
var _ = http.Flusher(&ResponseWriter{})

//
// Hijack a connection
//
func (w *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	} else {
		return nil, nil, errors.New("This ResponseWriter doesn't implement http.Hijacker")
	}
}

//
// Send any buffered data to the client
//
func (w *ResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

//
// Write response header
//
func (w *ResponseWriter) WriteHeader(status int) {
	w.wroteHeader = true

	var data []byte
	switch status / 100 {
	case 2:
		if w.OnSuccess != nil {
			w.OnSuccess(w.ResponseWriter, status)
		}
	case 4, 5:
		if w.OnError != nil {
			data = w.OnError(w.ResponseWriter, status)
		}
	}

	w.ResponseWriter.WriteHeader(status)
	if data != nil {
		w.skipBody = true
		w.ResponseWriter.Write(data)
	}
}

//
// Write response data
//
func (w *ResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if w.skipBody {
		return len(data), nil
	}

	return w.ResponseWriter.Write(data)
}
