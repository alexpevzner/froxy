//
// Tproxy instance
//

package main

import (
	"bytes"
	"cmd/tproxy/internal/pages"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

//
// tproxy instance
//
type Tproxy struct {
	*Env    // Common environment
	*Ebus   // Event bus
	*KeySet // Key set

	// Connection state
	connStateLock sync.Mutex // Access lock
	connState     ConnState  // Current state
	connStateInfo string     // Info string

	// Statistic counters
	Counters Counters // Collection of statistic counters

	// Tproxy parts
	router      *Router             // Request router
	webapi      *WebAPI             // JS API handler
	sysNotifier *SysNotifier        // System events notifier
	localhosts  map[string]struct{} // Hosts considered local
	localport   string              // Local port as string
	listener    net.Listener        // TCP listener
	httpSrv     *http.Server        // Local HTTP server instance

	// Transports
	sshTransport    *SSHTransport    // SSH transport
	directTransport *DirectTransport // Direct transport
	ftpProxy        *FTPProxy        // FTP-over-http proxy
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
		return "noconfig", "server not configured"
	case ConnTrying:
		return "trying", "trying..."
	case ConnEstablished:
		return "established", "connected to the server"
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

// ----- Connection management -----
//
// Set server parameters
//
func (proxy *Tproxy) SetServerParams(s ServerParams) {
	proxy.Env.SetServerParams(s)
	proxy.sshTransport.Reconnect(s)
}

// ----- Statistics counters -----
//
// Add value to the statistics counter
//
func (proxy *Tproxy) AddCounter(cnt *int32, val int32) {
	atomic.AddInt32(cnt, val)
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

	proxy.Debug("%s %s %s", r.Method, r.URL, r.Proto)

	// Strip hop-by-hop headers. Preserve Upgrade header, if any
	upgrade := httpRemoveHopByHopHeaders(r.Header)
	if upgrade {
		// FIXME
		//
		// Protocol upgrade is hard to implement with Go < 1.12.
		// Starting from Go 1.12 http.Response.Body implements
		// io.Writer interface, so protocol upgrade becomes simple
		// task.
		//
		// As for now, we explicitly reject upgrade requests.
		// Actually it's not a big problem, because browsers
		// implement websockets by calling proxy's CONNECT method
		// rather that GET with upgrade
		proxy.httpError(w, http.StatusServiceUnavailable,
			errors.New("Protocol upgrade is not implemented"))
	}

	// Perform round-trip
	resp, err := transport.RoundTrip(r)
	if err != nil {
		proxy.httpError(w, http.StatusServiceUnavailable, err)
		return
	}

	httpRemoveHopByHopHeaders(resp.Header)

	// Finish response, unless protocol is upgraded
	if resp.StatusCode != http.StatusSwitchingProtocols {
		proxy.returnHttpResponse(w, resp)
		return
	}

	// Handle protocol switch
	// TODO
}

//
// Return HTTP response back to the client
//
func (proxy *Tproxy) returnHttpResponse(w http.ResponseWriter, resp *http.Response) {
	httpCopyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	if resp.Body != nil {
		io.Copy(w, resp.Body)
		resp.Body.Close()
	}
}

// ----- Proxying CONNECT request -----
//
// HTTP CONNECT handler
//
func (proxy *Tproxy) handleConnect(
	w http.ResponseWriter,
	r *http.Request,
	transport Transport) {

	proxy.Debug("%s %s %s", r.Method, r.Host, r.Proto)

	dest_conn, err := transport.Dial("tcp", r.Host)
	if err != nil {
		proxy.httpError(w, http.StatusServiceUnavailable, err)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		proxy.httpError(w, http.StatusInternalServerError,
			errors.New("Hijacking not supported"))
		return
	}

	w.WriteHeader(http.StatusOK)
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		proxy.httpError(w, http.StatusServiceUnavailable, err)
		return
	}

	ioTransferData(proxy.Env, client_conn, dest_conn)
}

// ----- Handling requests to TProxy itself -----
//
// Handle local request
//
func (proxy *Tproxy) handleLocalRequest(w http.ResponseWriter, r *http.Request) {
	// Handle webapi requests
	if strings.HasPrefix(r.URL.Path, "/api/") {
		proxy.webapi.ServeHTTP(w, r)
		return
	}

	// Handle requests to TProxy static pages
	proxy.Debug("%s %s %s", r.Method, r.URL, r.Proto)
	httpNoCache(w)

	if r.URL.Path == "/" {
		//
		// TProxy home page is not very informative, so
		// it's better to redirect user to the last visited
		// page, if it is known, or to the configuration page
		// otherwise
		//
		url := "/conf/"
		if c, err := r.Cookie(COOKIE_LAST_VISITED_PAGE); err == nil {
			url = c.Value
		}

		http.Redirect(w, r, url, http.StatusFound)
		return
	}

	onsuccess := func(w http.ResponseWriter,
		status int) {
		proxy.httpOnSuccess(w, r, status)
	}

	w = &ResponseWriterWithHooks{
		ResponseWriter: w,
		OnError:        proxy.httpOnError,
		OnSuccess:      onsuccess,
	}

	// This allows CSS to be loaded when we
	// substitute a normal response with the
	// error page
	w.Header().Set(
		"Access-Control-Allow-Origin", "*")

	pages.FileServer.ServeHTTP(w, r)
}

// ----- HTTP response generation helpers -----
//
// Format body of error response
//
func (proxy *Tproxy) httpFormatError(status int, err error) (
	contentType string, content []byte) {

	// Fetch HTML error template
	var html []byte
	if file, err := pages.AssetFS.Open("error/index.html"); err == nil {
		html, err = ioutil.ReadAll(file)
		file.Close()

		if err != nil {
			html = nil
		}
	}

	// If no html, format simple text message
	if html == nil {
		s := fmt.Sprintf("%d %s\n", status, http.StatusText(status))
		if err != nil {
			s += fmt.Sprintf("%s\n", err)
		}

		contentType = "text/plain; charset=utf-8"
		content = []byte(s)
		return
	}

	// Substitute error information into HTML template
	contentType = "text/html; charset=utf-8"
	content = []byte(os.Expand(string(html), func(name string) string {
		switch name {
		case "ERROR":
			return http.StatusText(status)
		case "STATUS":
			return fmt.Sprintf("%d", status)
		case "MESSAGE":
			if err == nil {
				return ""
			} else {
				return err.Error()
			}
		}
		return ""
	}))

	return
}

//
// Canonicalize URLs, embedded into HTML document, by replacing
// relative links with absolute
//
func (proxy *Tproxy) httpCanonicalizeURLs(input []byte) []byte {
	return bytes.Replace(
		input,
		[]byte(`href="/`),
		[]byte(`href="http://`+proxy.httpSrv.Addr+"/"),
		-1,
	)
}

//
// Reply with HTTP error
//
func (proxy *Tproxy) httpError(w http.ResponseWriter, status int, err error) {
	contentType, content := proxy.httpFormatError(status, err)
	w.Header().Set("Content-Type", contentType)
	httpNoCache(w)

	w.WriteHeader(status)

	content = proxy.httpCanonicalizeURLs(content)
	w.Write(content)
}

//
// ResponseWriterWithHooks.OnError hook
//
func (proxy *Tproxy) httpOnError(w http.ResponseWriter, status int) []byte {
	contentType, content := proxy.httpFormatError(status, nil)
	w.Header().Set("Content-Type", contentType)
	httpNoCache(w)
	return content
}

//
// ResponseWriterWithHooks.OnSuccess hook
//
func (proxy *Tproxy) httpOnSuccess(w http.ResponseWriter,
	r *http.Request, status int) {

	path := r.URL.Path
	if path != "/" && strings.HasSuffix(path, "/") {
		http.SetCookie(w, &http.Cookie{
			Name:   COOKIE_LAST_VISITED_PAGE,
			Value:  r.URL.Path,
			Path:   "/",
			MaxAge: 365 * 24 * 60 * 60,
		})
	}
}

// ----- HTTP requests handling -----
//
// handle HTTP request. Provides multiplexing between regular request
// and CONNECT request handlers
//
func (proxy *Tproxy) httpHandler(w http.ResponseWriter, r *http.Request) {
	// Normalize hostname
	host, port := NetSplitHostPort(strings.ToLower(r.Host), "")

	// Check for request to TProxy itself
	if port == proxy.localport {
		_, local := proxy.localhosts[host]
		if local {
			proxy.handleLocalRequest(w, r)
			return
		}
	}

	// Check routing
	rt := proxy.router.Route(host)

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
		proxy.httpError(w, http.StatusForbidden, ErrSiteBlocked)
		return
	default:
		panic("internal error")
	}

	// Handle request
	switch {
	case r.URL.Scheme == "ftp":
		proxy.Debug("%s %s %s", r.Method, r.URL, r.Proto)
		proxy.ftpProxy.Handle(w, r, transport)
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
		switch e {
		case EventShutdownRequested:
			proxy.Debug("Shutdown requested. Exiting...")
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
		KeySet:     NewKeySet(env),
		localhosts: make(map[string]struct{}),
	}

	proxy.webapi = NewWebAPI(proxy)
	proxy.router = NewRouter(proxy)
	proxy.sysNotifier = NewSysNotifier(proxy)

	// Populate table of local host names
	for _, h := range []string{
		"localhost",
		"127.0.0.1",
		"127.1",
		"[::1]",
	} {
		proxy.localhosts[h] = struct{}{}
	}

	proxy.localport = fmt.Sprintf("%d", port)

	// Create transports
	proxy.sshTransport = NewSSHTransport(proxy)
	proxy.directTransport = NewDirectTransport(proxy)
	proxy.ftpProxy = NewFTPProxy(proxy)

	// Create HTTP server
	proxy.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: http.HandlerFunc(proxy.httpHandler),
	}

	// Create TCP listener
	var err error
	proxy.listener, err = NewListener(proxy, proxy.httpSrv.Addr)
	if err != nil {
		return nil, err
	}

	proxy.Info("Starting HTTP server at http://%s", proxy.httpSrv.Addr)

	// Update last used port
	if port != proxy.GetPort() {
		proxy.SetPort(port)
	}

	return proxy, nil
}
