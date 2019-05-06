//
// Froxy instance
//

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"cmd/froxy/internal/pages"
	"cmd/froxy/internal/sysdep"
)

//
// froxy instance
//
type Froxy struct {
	*Env    // Common environment
	*Ebus   // Event bus
	*KeySet // Key set

	// Connection state
	connStateLock sync.Mutex // Access lock
	connState     ConnState  // Current state
	connStateInfo string     // Info string

	// Statistic counters
	Counters Counters // Collection of statistic counters

	// Froxy parts
	router      *Router                  // Request router
	webapi      *WebAPI                  // JS API handler
	sysNotifier *sysdep.SysEventNotifier // System events notifier
	localhosts  map[string]struct{}      // Hosts considered local
	localport   string                   // Local port as string
	listener    net.Listener             // TCP listener
	httpSrv     *http.Server             // Local HTTP server instance

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
func (froxy *Froxy) GetConnState() (state ConnState, info string) {
	froxy.connStateLock.Lock()
	state = froxy.connState
	info = froxy.connStateInfo
	froxy.connStateLock.Unlock()

	return
}

//
// Set connection state
//
func (froxy *Froxy) SetConnState(state ConnState, info string) {
	froxy.connStateLock.Lock()

	if froxy.connState != state || froxy.connStateInfo != info {
		froxy.connState = state
		froxy.connStateInfo = info

		froxy.Raise(EventConnStateChanged)
	}

	froxy.connStateLock.Unlock()
}

// ----- Connection management -----
//
// Set server parameters
//
func (froxy *Froxy) SetServerParams(s ServerParams) {
	froxy.Env.SetServerParams(s)
	froxy.sshTransport.Reconnect(s)
}

// ----- Statistics counters -----
//
// Add value to the statistics counter
//
func (froxy *Froxy) AddCounter(cnt *int32, val int32) {
	atomic.AddInt32(cnt, val)
	froxy.Raise(EventCountersChanged)
}

//
// Increment the statistics counter
//
func (froxy *Froxy) IncCounter(cnt *int32) {
	froxy.AddCounter(cnt, 1)
}

//
// Decrement the statistics counter
//
func (froxy *Froxy) DecCounter(cnt *int32) {
	froxy.AddCounter(cnt, -1)
}

// ----- Proxying regular HTTP requests (GET/PUT/HEAD etc) -----
//
// Regular HTTP request handler
//
func (froxy *Froxy) handleRegularHttp(
	w http.ResponseWriter,
	r *http.Request,
	transport Transport) {

	froxy.Debug("%s %s %s", r.Method, r.URL, r.Proto)

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
		froxy.httpError(w, http.StatusServiceUnavailable,
			errors.New("Protocol upgrade is not implemented"))
	}

	// Perform round-trip
	resp, err := transport.RoundTrip(r)
	if err != nil {
		froxy.httpError(w, http.StatusServiceUnavailable, err)
		return
	}

	httpRemoveHopByHopHeaders(resp.Header)

	// Finish response, unless protocol is upgraded
	if resp.StatusCode != http.StatusSwitchingProtocols {
		froxy.returnHttpResponse(w, resp)
		return
	}

	// Handle protocol switch
	// TODO
}

//
// Return HTTP response back to the client
//
func (froxy *Froxy) returnHttpResponse(w http.ResponseWriter, resp *http.Response) {
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
func (froxy *Froxy) handleConnect(
	w http.ResponseWriter,
	r *http.Request,
	transport Transport) {

	froxy.Debug("%s %s %s", r.Method, r.Host, r.Proto)

	dest_conn, err := transport.Dial("tcp", r.Host)
	if err != nil {
		froxy.httpError(w, http.StatusServiceUnavailable, err)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		froxy.httpError(w, http.StatusInternalServerError,
			errors.New("Hijacking not supported"))
		return
	}

	w.WriteHeader(http.StatusOK)
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		froxy.httpError(w, http.StatusServiceUnavailable, err)
		return
	}

	ioTransferData(froxy.Env, client_conn, dest_conn)
}

// ----- Handling requests to Froxy itself -----
//
// Handle local request
//
func (froxy *Froxy) handleLocalRequest(w http.ResponseWriter, r *http.Request) {
	// Handle webapi requests
	if strings.HasPrefix(r.URL.Path, "/api/") {
		froxy.webapi.ServeHTTP(w, r)
		return
	}

	// Handle requests to Froxy static pages
	froxy.Debug("%s %s %s", r.Method, r.URL, r.Proto)
	httpNoCache(w)

	if r.URL.Path == "/" {
		//
		// Froxy home page is not very informative, so
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

	w = &ResponseWriterWithHooks{
		ResponseWriter: w,
		OnError:        froxy.httpOnError,
		OnSuccess: func(w http.ResponseWriter, status int) {
			froxy.httpOnSuccess(w, r, status)
		},
	}

	// We allow static content to be loaded from any
	// origin. This allows CSS to be loaded when we
	// substitute a normal response with the
	// error page
	w.Header().Set("Access-Control-Allow-Origin", "*")

	pages.FileServer.ServeHTTP(w, r)
}

// ----- HTTP response generation helpers -----
//
// Format body of error response
//
func (froxy *Froxy) httpFormatError(status int, err error) (
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
func (froxy *Froxy) httpCanonicalizeURLs(input []byte) []byte {
	return bytes.Replace(
		input,
		[]byte(`href="/`),
		[]byte(`href="`+froxy.BaseURL()),
		-1,
	)
}

//
// Reply with HTTP error
//
func (froxy *Froxy) httpError(w http.ResponseWriter, status int, err error) {
	contentType, content := froxy.httpFormatError(status, err)
	w.Header().Set("Content-Type", contentType)
	httpNoCache(w)

	w.WriteHeader(status)

	content = froxy.httpCanonicalizeURLs(content)
	w.Write(content)
}

//
// ResponseWriterWithHooks.OnError hook
//
func (froxy *Froxy) httpOnError(w http.ResponseWriter, status int) []byte {
	contentType, content := froxy.httpFormatError(status, nil)
	w.Header().Set("Content-Type", contentType)
	httpNoCache(w)
	return content
}

//
// ResponseWriterWithHooks.OnSuccess hook
//
func (froxy *Froxy) httpOnSuccess(w http.ResponseWriter,
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
func (froxy *Froxy) httpHandler(w http.ResponseWriter, r *http.Request) {
	// Normalize hostname
	host, port := NetSplitHostPort(strings.ToLower(r.Host), "")

	// Check for request to Froxy itself
	if port == froxy.localport {
		_, local := froxy.localhosts[host]
		if local {
			froxy.handleLocalRequest(w, r)
			return
		}
	}

	// Check routing
	rt := froxy.router.Route(host)

	// Update counters
	froxy.IncCounter(&froxy.Counters.HTTPRqReceived)
	froxy.IncCounter(&froxy.Counters.HTTPRqPending)
	defer froxy.DecCounter(&froxy.Counters.HTTPRqPending)

	// Choose transport
	var transport Transport
	switch rt {
	case RouterBypass:
		froxy.IncCounter(&froxy.Counters.HTTPRqDirect)
		transport = froxy.directTransport
	case RouterForward:
		froxy.IncCounter(&froxy.Counters.HTTPRqForwarded)
		transport = froxy.sshTransport
	case RouterBlock:
		froxy.IncCounter(&froxy.Counters.HTTPRqBlocked)
		froxy.httpError(w, http.StatusForbidden, ErrSiteBlocked)
		return
	default:
		panic("internal error")
	}

	// Handle request
	switch {
	case r.URL.Scheme == "ftp":
		froxy.Debug("%s %s %s", r.Method, r.URL, r.Proto)
		froxy.ftpProxy.Handle(w, r, transport)
	case r.Method == http.MethodConnect:
		froxy.handleConnect(w, r, transport)
	default:
		froxy.handleRegularHttp(w, r, transport)
	}
}

// ----- Events handling -----
//
// Event monitoring goroutine
//
func (froxy *Froxy) eventGoroutine() {
	events := froxy.Sub()
	for {
		e := <-events
		switch e {
		case EventShutdownRequested:
			froxy.Debug("Shutdown requested. Exiting...")
			os.Exit(0)
		}
	}
}

//
// sysdep.SysEventNotifier callback
//
func (froxy *Froxy) sysEventCallback(se sysdep.SysEvent) {
	switch se {
	case sysdep.SysEventShutdown:
		froxy.Info("Shutdown requested")
		froxy.Raise(EventShutdownRequested)
	}
}

// ----- Miscellaneous helpers -----
//
// Get Froxy base URL (i.e., "http://localhost:8888/"
//
func (froxy *Froxy) BaseURL() string {
	return "http://" + froxy.httpSrv.Addr + "/"
}

// ----- Froxy initialization -----
//
// Create a Froxy instance
//
func NewFroxy(env *Env, port int) (*Froxy, error) {
	// Create Froxy structure
	froxy := &Froxy{
		Env:        env,
		Ebus:       NewEbus(),
		KeySet:     NewKeySet(env),
		localhosts: make(map[string]struct{}),
	}

	froxy.webapi = NewWebAPI(froxy)
	froxy.router = NewRouter(froxy)
	froxy.sysNotifier = sysdep.NewSysEventNotifier(froxy.sysEventCallback)

	// Populate table of local host names
	for _, h := range []string{
		"localhost",
		"127.0.0.1",
		"127.1",
		"[::1]",
	} {
		froxy.localhosts[h] = struct{}{}
	}

	froxy.localport = fmt.Sprintf("%d", port)

	// Create transports
	froxy.sshTransport = NewSSHTransport(froxy)
	froxy.directTransport = NewDirectTransport(froxy)
	froxy.ftpProxy = NewFTPProxy(froxy)

	// Create HTTP server
	froxy.httpSrv = &http.Server{
		Addr:     fmt.Sprintf("localhost:%d", port),
		Handler:  http.HandlerFunc(froxy.httpHandler),
		ErrorLog: log.New(froxy.NewLogWriter(LogLevelError), "", 0),
	}

	// Create TCP listener
	var err error
	froxy.listener, err = NewListener(froxy, froxy.httpSrv.Addr)
	if err != nil {
		return nil, err
	}

	froxy.Info("Starting HTTP server at http://%s", froxy.httpSrv.Addr)

	// Update last used port
	if port != froxy.GetPort() {
		froxy.SetPort(port)
	}

	return froxy, nil
}

//
// Run Froxy
//
func (froxy *Froxy) Run() {
	go froxy.eventGoroutine()
	froxy.Raise(EventStartup)

	err := froxy.httpSrv.Serve(froxy.listener)
	if err != nil {
		panic("Internal error: " + err.Error())
	}
}
