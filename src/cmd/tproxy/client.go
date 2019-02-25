//
// Client-side proxy
//

package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
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
func (proxy *tproxyClient) handleRegularHttp(w http.ResponseWriter, r *http.Request) {
	forward := proxy.router.Route(r.URL)
	log.Debug("forward=%v", forward)
	log.Debug("host=%v", r.Host)

	httpRemoveHopByHopHeaders(r.Header)
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

	dump, _ := httputil.DumpRequest(r, false)
	log.Debug("===== request =====\n%s", dump)

	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		log.Debug("  %s", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	if forward {
		if resp.StatusCode == http.StatusUnauthorized {
			h := resp.Header.Get("X-Tproxy-Authenticate")
			if h != "" {
				resp.StatusCode = http.StatusProxyAuthRequired
				resp.Header.Set("Proxy-Authenticate", h)
			}
		}
	}

	dump, _ = httputil.DumpResponse(resp, false)
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
func (proxy *tproxyClient) handleConnect(w http.ResponseWriter, r *http.Request) {
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
	go proxy.transferData(dest_conn, client_conn)
	go proxy.transferData(client_conn, dest_conn)
}

//
// Transfer data between two connections
//
func (proxy *tproxyClient) transferData(destination io.WriteCloser, source io.ReadCloser) {
	io.Copy(destination, source)
	destination.Close()
	source.Close()
}

//
// handle HTTP request. Provides multiplexing between regular request
// and CONNECT request handlers
//
func (proxy *tproxyClient) httpHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("%s %s %s", r.Proto, r.Method, r.URL)

	// Handle request
	if r.Method == http.MethodConnect {
		proxy.handleConnect(w, r)
	} else {
		proxy.handleRegularHttp(w, r)
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
		Addr:    fmt.Sprintf("127.1:%d", cfg.Port),
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
