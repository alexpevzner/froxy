//
// JS API for configuration pages
//

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

//
// JS API handler
//
type WebAPI struct {
	env *Env
	mux *http.ServeMux
}

var _ = http.Handler(&WebAPI{})

//
// Create new JS API handler instance
//
func NewWebAPI(env *Env) *WebAPI {
	webapi := &WebAPI{
		env: env,
		mux: http.NewServeMux(),
	}

	webapi.mux.HandleFunc("/server", webapi.handleServer)
	webapi.mux.HandleFunc("/sites", webapi.handleSites)

	return webapi
}

//
// Handle /server requests
//
func (webapi *WebAPI) handleServer(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		conf := webapi.env.GetServerParams()
		webapi.replyJSON(w, conf)
	default:
		webapi.httpError(w, http.StatusMethodNotAllowed)
	}
}

//
// Handle /sites requests
//
func (webapi *WebAPI) handleSites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		conf := webapi.env.GetSites()
		webapi.replyJSON(w, conf)
	default:
		webapi.httpError(w, http.StatusMethodNotAllowed)
	}
}

//
// Handle HTTP request
//
func (webapi *WebAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/conf":
	case "/sites":
	}

	webapi.mux.ServeHTTP(w, r)
}

//
// Return HTTP error
//
func (webapi *WebAPI) httpError(w http.ResponseWriter, status int) {
	http.Error(w, fmt.Sprintf("%d %s", status, http.StatusText(status)), status)
}

//
// Reply with JSON data
//
func (webapi *WebAPI) replyJSON(w http.ResponseWriter, data interface{}) {
	body, _ := json.Marshal(data)
	w.Header().Set("Content-Type", "application/json")

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	w.Write(body)
	w.Write([]byte("\n"))
}
