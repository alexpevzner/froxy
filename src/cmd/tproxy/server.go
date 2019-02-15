//
// tproxy server-side proxy
//

package main

import (
	"fmt"
	"net/http"
	"tproxy/log"
)

//
// tproxy server
//
type tproxyServer struct {
	cfg     *CfgServer   // Server configuration
	httpSrv *http.Server // HTTP server instance
}

//
// HTTP request handler
//
func (server *tproxyServer) httpHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusInternalServerError)
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
	// Load configiration file
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
	}

	server.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("127.1:%d", cfg.Port),
		Handler: http.HandlerFunc(server.httpHandler),
	}

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
