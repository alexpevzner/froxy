// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// A collection of useful http.ResponseWriter implementations

package main

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"net/http"
)

// ----- ResponseWriterWithHooks -----
//
// http.ResponseWriter wrapper with hooks on request completion
//
type ResponseWriterWithHooks struct {
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

	wroteHeader bool // Header was written
	skipBody    bool // Skip the body
}

var _ = http.ResponseWriter(&ResponseWriterWithHooks{})
var _ = http.Hijacker(&ResponseWriterWithHooks{})
var _ = http.Flusher(&ResponseWriterWithHooks{})

//
// Hijack a connection
//
func (w *ResponseWriterWithHooks) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	} else {
		return nil, nil, errors.New("This ResponseWriter doesn't implement http.Hijacker")
	}
}

//
// Send any buffered data to the client
//
func (w *ResponseWriterWithHooks) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

//
// Write response header
//
func (w *ResponseWriterWithHooks) WriteHeader(status int) {
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
func (w *ResponseWriterWithHooks) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if w.skipBody {
		return len(data), nil
	}

	return w.ResponseWriter.Write(data)
}

// ----- ResponseWriterWithBuffer -----
//
// This ResponseWriter saves response bytes into its own
// buffer instead of sending them to client
//
type ResponseWriterWithBuffer struct {
	bytes.Buffer             // Underlying buffer
	Status       int         // Response HTTP status
	header       http.Header // Response header
	wroteHeader  bool        // Header was written
}

var _ = http.ResponseWriter(&ResponseWriterWithBuffer{})

//
// Get response header
//
func (w *ResponseWriterWithBuffer) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

//
// Write response header
//
func (w *ResponseWriterWithBuffer) WriteHeader(status int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.Status = status
	}
}

//
// Write response data
//
func (w *ResponseWriterWithBuffer) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.Buffer.Write(data)
}

//
// Send collected response to http.ResponseWriter
//
func (w *ResponseWriterWithBuffer) Send(to http.ResponseWriter) {
	if w.Status == 0 {
		w.WriteHeader(http.StatusOK)
	}

	httpCopyHeaders(to.Header(), w.Header())
	to.WriteHeader(w.Status)
	to.Write(w.Bytes())
}

//
// Reset the ResponseWriterWithBuffer
//
func (w *ResponseWriterWithBuffer) Reset() {
	// Reset all but Buffer and header
	*w = ResponseWriterWithBuffer{Buffer: w.Buffer, header: w.header}

	// Reset Buffer and header
	w.Buffer.Reset()
	for k := range w.header {
		delete(w.header, k)
	}
}
