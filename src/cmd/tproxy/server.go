//
// tproxy server-side proxy
//

package main

import (
	"crypto/subtle"
	"fmt"
	"io"
	"net/http"
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
	http.Error(w, "Not implemented", http.StatusInternalServerError)
}

//
// /exec handler - executes regular HTTP requests
//
func (server *tproxyServer) httpExecHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusInternalServerError)
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
	// Handle user authentication
	user, password, ok := r.BasicAuth()
	if !ok || len(user) == 0 || len(password) == 0 {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Authentication Required", http.StatusUnauthorized)
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
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	// Now forward request to multiplexer
	server.mux.ServeHTTP(w, r)
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
