//
// tproxy server-side proxy
//

package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"tproxy/log"
)

//
// tproxy server
//
type tproxyServer struct {
	cfg     *CfgServer     // Server configuration
	httpSrv *http.Server   // HTTP server instance
	mux     *http.ServeMux // request multiplexer
}

//
// /conn handler - handles CONNECT requests
//
func (server *tproxyServer) httpConnHandler(w http.ResponseWriter, r *http.Request) {
	// Establish destination connection
	dest_conn, err := net.DialTimeout("tcp", r.URL.RawQuery, CONNECT_TIMEOUT)

	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Switch to websocket mode
	ws, err := websockUpgrade(w, r)
	if err != nil {
		return
	}

	ioTransferData(ws, dest_conn)
}

//
// /exec handler - executes regular HTTP requests
//
func (server *tproxyServer) httpExecHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	r.URL, err = url.Parse(r.URL.RawQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	r.Host = r.Header.Get("X-Tproxy-Host")
	r.Header.Del("X-Tproxy-Host")
	r.Header.Del("X-Tproxy-Authorization")

	httpRemoveHopByHopHeaders(r.Header)
	r.RequestURI = ""

	dump, _ := httputil.DumpRequest(r, false)
	log.Debug("===== decoded request =====\n%s", dump)

	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		log.Debug("  %s", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	dump, _ = httputil.DumpResponse(resp, false)
	log.Debug("===== response =====\n%s", dump)

	httpCopyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	resp.Body.Close()
}

//
// /auth handler - does nothing, purposed to validate authentication
//
func (server *tproxyServer) httpAuthHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK\n")
}

//
// HTTP request handler
//
func (server *tproxyServer) httpHandler(w http.ResponseWriter, r *http.Request) {
	dump, _ := httputil.DumpRequest(r, false)
	log.Debug("===== request =====\n%s", dump)

	// Handle user authentication
	user, password, ok := server.getProxyUserPassword(r)
	if !ok || len(user) == 0 || len(password) == 0 {
		server.statusUnauthorized(w, r)
		return
	}

	// Attempt to compare passwords in constant time
	//
	// FIXME - make this stuff allocation-free
	pwd_bytes := []byte(password + "X")
	good_pwd_bytes := server.cfg.Users[user] + "X"

	good_pwd2_bytes := make([]byte, len(pwd_bytes))
	for i := 0; i < len(pwd_bytes); i++ {
		good_pwd2_bytes[i] = good_pwd_bytes[i%len(good_pwd_bytes)]
	}

	ok = subtle.ConstantTimeCompare(pwd_bytes, good_pwd2_bytes) == 1 &&
		len(pwd_bytes) == len(good_pwd_bytes)

	if !ok {
		server.statusUnauthorized(w, r)
		return
	}

	// Now forward request to multiplexer
	server.mux.ServeHTTP(w, r)
}

//
// Get proxy authentication credentials
//
func (server *tproxyServer) getProxyUserPassword(r *http.Request) (user, password string, ok bool) {
	auth := r.Header.Get("X-Tproxy-Authorization")
	if auth == "" {
		return
	}

	log.Debug("auth=%s", auth)

	const prefix = "Basic "
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}

	log.Debug("decoded=%s", decoded)
	sep := bytes.IndexByte(decoded, ':')
	if sep < 0 {
		return
	}

	return string(decoded[:sep]), string(decoded[sep+1:]), true
}

//
// Finish request with the http.StatusUnauthorized response
//
func (server *tproxyServer) statusUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Tproxy-Authenticate", `Basic realm="Tproxy authentication required"`)
	http.Error(w, "Not authorized", http.StatusUnauthorized)
}

//
// Run a server
//
func (server *tproxyServer) Run() error {
	return server.httpSrv.ListenAndServe()
}

//
// Create a server
//
func newTproxyServer(cfgPath string) (*tproxyServer, error) {
	// Load configuration file
	if cfgPath == "" {
		cfgPath = DEFAULT_SERVER_CFG
	}

	cfg, err := LoadCfgServer(cfgPath)
	if err != nil {
		return nil, err
	}

	// Create tproxyServer structure
	server := &tproxyServer{
		cfg: cfg,
		mux: http.NewServeMux(),
	}

	server.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("127.1:%d", cfg.Port),
		Handler: http.HandlerFunc(server.httpHandler),
	}

	// Setup request multiplexer
	server.mux.HandleFunc("/auth", server.httpAuthHandler)
	server.mux.HandleFunc("/exec", server.httpExecHandler)
	server.mux.HandleFunc("/conn", server.httpConnHandler)

	return server, nil
}

//
// Run tproxy in a server mode
//
func runServer(cfgPath string) {
	proxy, err := newTproxyServer(cfgPath)
	if err == nil {
		err = proxy.Run()
	}

	if err != nil {
		log.Exit("%s", err)
	}
}
