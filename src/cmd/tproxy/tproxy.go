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
	env             *Env                // Common environment
	router          *Router             // Request router
	webapi          *WebAPI             // JS API handler
	localhosts      map[string]struct{} // Hosts considered local
	listener        net.Listener        // TCP listener
	httpSrv         *http.Server        // Local HTTP server instance
	sshTransport    *SSHTransport       // SSH transport
	directTransport *DirectTransport    // Direct transport
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
		httpErrorf(w, http.StatusServiceUnavailable, "%s", err)
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
		httpErrorf(w, http.StatusServiceUnavailable, "%s", err)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		httpErrorf(w, http.StatusInternalServerError, "Hijacking not supported")
		return
	}

	w.WriteHeader(http.StatusOK)
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		httpErrorf(w, http.StatusServiceUnavailable, "%s", err)
		return
	}

	ioTransferData(proxy.env, client_conn, dest_conn)
}

//
// handle HTTP request. Provides multiplexing between regular request
// and CONNECT request handlers
//
func (proxy *Tproxy) httpHandler(w http.ResponseWriter, r *http.Request) {
	proxy.env.Debug("%s %s %s", r.Method, r.URL, r.Proto)

	// Normalize hostname
	host := strings.ToLower(r.Host)

	// Check for request to TProxy itself
	_, local := proxy.localhosts[r.Host]
	if local {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			dump, _ := httputil.DumpRequest(r, false)
			proxy.env.Debug("===== request =====\n%s", dump)
			proxy.env.Debug("local->webapi")
			proxy.webapi.ServeHTTP(w, r)
		} else {
			proxy.env.Debug("local->site")
			pages.FileServer.ServeHTTP(w, r)
		}
		return
	}

	// Check routing
	host, _ = NetSplitHostPort(strings.ToLower(r.Host), "")
	if host == HTTP_SERVER_HOST {
		// HTTP_SERVER_HOST attempted with invalid port
		httpErrorf(w, http.StatusServiceUnavailable, "Invalid port")
		return
	}

	rt := proxy.router.Route(host)
	proxy.env.Debug("router answer=%s", rt)
	proxy.env.Debug("host=%v", r.Host)

	// Update counters
	proxy.env.IncCounter(&proxy.env.Counters.HTTPRqReceived)
	proxy.env.IncCounter(&proxy.env.Counters.HTTPRqPending)
	defer proxy.env.DecCounter(&proxy.env.Counters.HTTPRqPending)

	// Choose transport
	var transport Transport
	switch rt {
	case RouterBypass:
		proxy.env.IncCounter(&proxy.env.Counters.HTTPRqDirect)
		transport = proxy.directTransport
	case RouterForward:
		proxy.env.IncCounter(&proxy.env.Counters.HTTPRqForwarded)
		transport = proxy.sshTransport
	case RouterBlock:
		proxy.env.IncCounter(&proxy.env.Counters.HTTPRqBlocked)
		httpErrorf(w, http.StatusForbidden, "Site blocked")
		return
	default:
		panic("internal error")
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
func (proxy *Tproxy) Run() {
	err := proxy.httpSrv.Serve(proxy.listener)
	if err != nil {
		panic("Internal error: " + err.Error())
	}
}

//
// Create a Tproxy instance
//
func NewTproxy(port int) (*Tproxy, error) {
	// Create Tproxy structure
	env := NewEnv()
	proxy := &Tproxy{
		env:        env,
		router:     NewRouter(env),
		webapi:     NewWebAPI(env),
		localhosts: make(map[string]struct{}),
	}

	// Populate table of local host names
	for _, h := range []string{
		"localhost",
		"127.0.0.1",
		"127.1",
		"[::1]",
		HTTP_SERVER_HOST,
	} {
		hp := fmt.Sprintf("%s:%d", h, port)
		proxy.localhosts[hp] = struct{}{}
	}

	proxy.localhosts[HTTP_SERVER_HOST] = struct{}{}

	// Create transports
	proxy.sshTransport = NewSSHTransport(env)
	proxy.directTransport = NewDirectTransport(env)

	// Create HTTP server
	proxy.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: http.HandlerFunc(proxy.httpHandler),
	}

	// Create TCP listener
	var err error
	proxy.listener, err = net.Listen("tcp", proxy.httpSrv.Addr)
	if err != nil {
		return nil, err
	}

	env.Debug("Starting HTTP server at http://%s", proxy.httpSrv.Addr)

	return proxy, nil
}
