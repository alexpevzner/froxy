//
// Client-side proxy
//

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"tproxy/log"
)

//
// tproxy client
//
type tproxyClient struct {
	cfg     *CfgClient   // Client configuration
	httpSrv *http.Server // Local HTTP server instance
	router  *Router      // Request router
}

// ----- Proxying regular HTTP requests (GET/PUT/HEAD etc) -----
//
// Regular HTTP request handler
//
func (proxy *tproxyClient) handleRegularHttp(
	w http.ResponseWriter,
	r *http.Request,
	transport Transport) {

	httpRemoveHopByHopHeaders(r.Header)
	/*
		if forward {
			r.URL, _ = url.Parse(proxy.cfg.Server.String() + "/exec?" + r.URL.String())
			r.Header.Set("X-Tproxy-Host", r.Host)
			r.Host = proxy.cfg.Server.Host

			h := r.Header.Get("Proxy-Authorization")
			if h != "" {
				r.Header.Set("X-Tproxy-Authorization", h)
				r.Header.Del("Proxy-Authorization")
			}
		}
	*/

	dump, _ := httputil.DumpRequest(r, false)
	log.Debug("===== request =====\n%s", dump)

	resp, err := transport.RoundTrip(r)

	if err != nil {
		log.Debug("  %s", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	proxy.returnHttpResponse(w, resp)
}

//
// Return HTTP response back to the client
//
func (proxy *tproxyClient) returnHttpResponse(w http.ResponseWriter, resp *http.Response) {
	dump, _ := httputil.DumpResponse(resp, false)
	log.Debug("===== response =====\n%s", dump)

	httpCopyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	resp.Body.Close()
}

// ----- Proxying CONNECT request -----
//
// HTTP CONNECT handler
//
func (proxy *tproxyClient) handleConnect(
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
func (proxy *tproxyClient) httpHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("%s %s %s", r.Method, r.URL, r.Proto)

	forward := proxy.router.Route(r.URL)
	log.Debug("forward=%v", forward)
	log.Debug("host=%v", r.Host)

	var transport Transport
	if forward {
		transport = sshTransport
	} else {
		transport = directTransport
	}

	// Handle request
	if r.Method == http.MethodConnect {
		proxy.handleConnect(w, r, transport)
	} else {
		proxy.handleRegularHttp(w, r, transport)
	}
}

//
// Run a client proxy
//
func (proxy *tproxyClient) Run() error {
	return proxy.httpSrv.ListenAndServe()
}

//
// Create a client
//
func newTproxyClient(cfgPath string) (*tproxyClient, error) {
	// Load configiration file
	if cfgPath == "" {
		cfgPath = DEFAULT_CLIENT_CFG
	}

	cfg, err := LoadCfgClient(cfgPath)
	if err != nil {
		return nil, err
	}

	// Create tproxyClient structure
	proxy := &tproxyClient{
		cfg:    cfg,
		router: NewRouter(),
	}

	proxy.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", cfg.Port),
		Handler: http.HandlerFunc(proxy.httpHandler),
	}

	proxy.router.SetSites(cfg.Sites)

	return proxy, nil
}

//
// Run tproxy in a client mode
//
func runClient(cfgPath string) {
	proxy, err := newTproxyClient(cfgPath)
	if err == nil {
		err = proxy.Run()
	}

	if err != nil {
		log.Exit("%s", err)
	}
}
