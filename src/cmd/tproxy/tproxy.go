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
	"os"
	"pages"
	"strings"
	"sync"
	"sync/atomic"
)

//
// tproxy instance
//
type Tproxy struct {
	*Env  // Common environment
	*Ebus // Event bus

	// Connection state
	connStateLock sync.Mutex // Access lock
	connState     ConnState  // Current state
	connStateInfo string     // Info string

	// Statistic counters
	Counters Counters // Collection of statistic counters

	// Tproxy parts
	router     *Router             // Request router
	webapi     *WebAPI             // JS API handler
	localhosts map[string]struct{} // Hosts considered local
	listener   net.Listener        // TCP listener
	httpSrv    *http.Server        // Local HTTP server instance

	// Transports
	sshTransport    *SSHTransport    // SSH transport
	directTransport *DirectTransport // Direct transport
}

// ----- Connection state -----
//
// Connection states
//
type ConnState int

const (
	ConnNotConfigured = ConnState(iota)
	ConnTrying
	ConnEstablished
)

//
// ConnState -> ("name", "default info string")
//
func (s ConnState) Strings() (string, string) {
	switch s {
	case ConnNotConfigured:
		return "noconfig", "Server not configured"
	case ConnTrying:
		return "trying", ""
	case ConnEstablished:
		return "established", "Connected to the server"
	}

	panic("internal error")
}

//
// Get connection state
//
func (proxy *Tproxy) GetConnState() (state ConnState, info string) {
	proxy.connStateLock.Lock()
	state = proxy.connState
	info = proxy.connStateInfo
	proxy.connStateLock.Unlock()

	return
}

//
// Set connection state
//
func (proxy *Tproxy) SetConnState(state ConnState, info string) {
	proxy.connStateLock.Lock()

	if proxy.connState != state {
		proxy.connState = state
		proxy.connStateInfo = info

		proxy.Raise(EventConnStateChanged)
	}

	proxy.connStateLock.Unlock()
}

// ----- Statistics counters -----
//
// Add value to the statistics counter
//
func (proxy *Tproxy) AddCounter(cnt *int32, val int32) {
	atomic.AddInt32(cnt, val)
	atomic.AddUint64(&proxy.Counters.Tag, 1)
	proxy.Raise(EventCountersChanged)
}

//
// Increment the statistics counter
//
func (proxy *Tproxy) IncCounter(cnt *int32) {
	proxy.AddCounter(cnt, 1)
}

//
// Decrement the statistics counter
//
func (proxy *Tproxy) DecCounter(cnt *int32) {
	proxy.AddCounter(cnt, -1)
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
	proxy.Debug("===== request =====\n%s", dump)

	resp, err := transport.RoundTrip(r)

	if err != nil {
		proxy.Debug("  %s", err)
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
	proxy.Debug("===== response =====\n%s", dump)

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

	ioTransferData(proxy.Env, client_conn, dest_conn)
}

//
// handle HTTP request. Provides multiplexing between regular request
// and CONNECT request handlers
//
func (proxy *Tproxy) httpHandler(w http.ResponseWriter, r *http.Request) {
	proxy.Debug("%s %s %s", r.Method, r.URL, r.Proto)

	// Normalize hostname
	host := strings.ToLower(r.Host)

	// Check for request to TProxy itself
	_, local := proxy.localhosts[r.Host]
	if local {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			dump, _ := httputil.DumpRequest(r, false)
			proxy.Debug("===== request =====\n%s", dump)
			proxy.Debug("local->webapi")
			proxy.webapi.ServeHTTP(w, r)
		} else {
			proxy.Debug("local->site")
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
	proxy.Debug("router answer=%s", rt)
	proxy.Debug("host=%v", r.Host)

	// Update counters
	proxy.IncCounter(&proxy.Counters.HTTPRqReceived)
	proxy.IncCounter(&proxy.Counters.HTTPRqPending)
	defer proxy.DecCounter(&proxy.Counters.HTTPRqPending)

	// Choose transport
	var transport Transport
	switch rt {
	case RouterBypass:
		proxy.IncCounter(&proxy.Counters.HTTPRqDirect)
		transport = proxy.directTransport
	case RouterForward:
		proxy.IncCounter(&proxy.Counters.HTTPRqForwarded)
		transport = proxy.sshTransport
	case RouterBlock:
		proxy.IncCounter(&proxy.Counters.HTTPRqBlocked)
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
	go proxy.eventGoroutine()
	proxy.Raise(EventStartup)

	err := proxy.httpSrv.Serve(proxy.listener)
	if err != nil {
		panic("Internal error: " + err.Error())
	}
}

//
// Event monitoring goroutine
//
func (proxy *Tproxy) eventGoroutine() {
	events := proxy.Sub()
	for {
		e := <-events
		proxy.Debug("%s", e)
		switch e {
		case EventShutdownRequested:
			os.Exit(0)
		}
	}
}

//
// Create a Tproxy instance
//
func NewTproxy(env *Env, port int) (*Tproxy, error) {
	// Create Tproxy structure
	proxy := &Tproxy{
		Env:        env,
		Ebus:       NewEbus(),
		localhosts: make(map[string]struct{}),
	}

	proxy.webapi = NewWebAPI(proxy)
	proxy.router = NewRouter(proxy)

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
	proxy.sshTransport = NewSSHTransport(proxy)
	proxy.directTransport = NewDirectTransport(proxy)

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

	proxy.Debug("Starting HTTP server at http://%s", proxy.httpSrv.Addr)

	// Update last used port
	if port != proxy.GetPort() {
		proxy.SetPort(port)
	}

	return proxy, nil
}
