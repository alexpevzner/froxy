//
// Tproxy instance
//

package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"pages"
	"strings"
)

//
// tproxy instance
//
type Tproxy struct {
	env     *Env         // Common environment
	cfg     *CfgTproxy   // Tproxy configuration
	httpSrv *http.Server // Local HTTP server instance
	router  *Router      // Request router
	webapi  *WebAPI      // JS API handler
}

// ----- Proxying regular HTTP requests (GET/PUT/HEAD etc) -----
//
// Regular HTTP request handler
//
func (proxy *Tproxy) handleRegularHttp(
	w http.ResponseWriter,
	r *http.Request,
	transport Transport) {

	httpRemoveHopByHopHeaders(r.Header)

	dump, _ := httputil.DumpRequest(r, false)
	proxy.env.Debug("===== request =====\n%s", dump)

	resp, err := transport.RoundTrip(r)

	if err != nil {
		proxy.env.Debug("  %s", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	proxy.returnHttpResponse(w, resp)
}

//
// Return HTTP response back to the client
//
func (proxy *Tproxy) returnHttpResponse(w http.ResponseWriter, resp *http.Response) {
	dump, _ := httputil.DumpResponse(resp, false)
	proxy.env.Debug("===== response =====\n%s", dump)

	httpCopyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	resp.Body.Close()
}

// ----- Proxying CONNECT request -----
//
// HTTP CONNECT handler
//
func (proxy *Tproxy) handleConnect(
	w http.ResponseWriter,
	r *http.Request,
	transport Transport) {

	dest_conn, err := transport.Dial("tcp", r.Host)
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

	ioTransferData(client_conn, dest_conn)
}

//
// handle HTTP request. Provides multiplexing between regular request
// and CONNECT request handlers
//
func (proxy *Tproxy) httpHandler(w http.ResponseWriter, r *http.Request) {
	proxy.env.Debug("%s %s %s", r.Method, r.URL, r.Proto)

	// Split host and port
	host := strings.ToLower(r.Host)

	if strings.IndexByte(host, ':') != -1 {
		h, _, err := net.SplitHostPort(host)
		if err == nil {
			host = h
		}
	}

	// Check routing
	forward := proxy.router.Route(host)
	proxy.env.Debug("forward=%v", forward)
	proxy.env.Debug("host=%v", r.Host)

	var transport Transport
	if forward {
		transport = sshTransport
	} else {
		transport = directTransport
	}

	// Check for request to TProxy itself
	switch host {
	case HOST_TPROXY_PAGES, "localhost":
		if strings.HasPrefix(r.URL.Path, "/api/") {
			proxy.env.Debug("local->webapi")
			proxy.webapi.ServeHTTP(w, r)
		} else {
			proxy.env.Debug("local->site")
			pages.FileServer.ServeHTTP(w, r)
		}
		return
	}

	// Handle request
	switch {
	case r.Method == http.MethodConnect:
		proxy.handleConnect(w, r, transport)
	default:
		proxy.handleRegularHttp(w, r, transport)
	}
}

//
// Run a proxy
//
func (proxy *Tproxy) Run() error {
	return proxy.httpSrv.ListenAndServe()
}

//
// Create a Tproxy instance
//
func NewTproxy(cfgPath string) (*Tproxy, error) {
	// Load configiration file
	if cfgPath == "" {
		cfgPath = DEFAULT_TPROXY_CFG
	}

	cfg, err := LoadCfg(cfgPath)
	if err != nil {
		return nil, err
	}

	// Create Tproxy structure
	env := NewEnv()
	proxy := &Tproxy{
		env:    env,
		cfg:    cfg,
		router: NewRouter(env),
		webapi: NewWebAPI(env),
	}

	proxy.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", cfg.Port),
		Handler: http.HandlerFunc(proxy.httpHandler),
	}

	return proxy, nil
}
