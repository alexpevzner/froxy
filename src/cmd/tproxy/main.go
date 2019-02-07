package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"tproxy/log"
)

// ----- Proxying regular HTTP requests (GET/PUT/HEAD etc) -----
//
// Regular HTTP request handler
//
func proxyHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		log.Debug("  %s", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	resp.Body.Close()
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// ----- Proxying CONNECT request -----
//
// HTTP CONNECT handler
//
func proxyConnect(w http.ResponseWriter, r *http.Request) {
	dest_conn, err := net.DialTimeout("tcp", r.Host, CONNECT_TIMEOUT)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	io.Copy(destination, source)
	destination.Close()
	source.Close()
}

//
// handle HTTP request. Provides multiplexing between regular request
// and CONNECT request handlers
//
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("%s %s %s", r.Proto, r.Method, r.URL)

	if r.Method == http.MethodConnect {
		proxyConnect(w, r)
	} else {
		proxyHTTP(w, r)
	}
}

func main() {
	server := &http.Server{
		Addr:    fmt.Sprintf("127.1:%d", LISTEN_PORT),
		Handler: http.HandlerFunc(proxyHandler),
	}

	s, e := LoadCfgServer("server.cfg")
	log.Debug("server.cfg: %#v %s", s, e)

	c, e := LoadCfgClient("client.cfg")
	log.Debug("client.cfg: %#v %s", c, e)

	err := server.ListenAndServe()
	if err != nil {
		log.Exit("%s", err)
	} else {
		log.Debug("Running proxy at %s", server.Addr)
	}
}
