//
// JS API for configuration pages
//

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	webapi.mux.HandleFunc("/api/server", webapi.handleServer)
	webapi.mux.HandleFunc("/api/sites", webapi.handleSites)

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

	case "PUT":
		body, err := ioutil.ReadAll(r.Body)
		var data StateServer

		if err != nil {
			goto FAIL
		}

		err = json.Unmarshal(body, &data)
		if err != nil {
			goto FAIL
		}

		webapi.env.SetServerParams(&data)
		return

	FAIL:
		webapi.httpErrorf(w, http.StatusInternalServerError, "%s", err)

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

	case "PUT":
		body, err := ioutil.ReadAll(r.Body)
		var data StateSite

		if err != nil {
			goto FAIL
		}

		err = json.Unmarshal(body, &data)
		if err != nil {
			goto FAIL
		}

		webapi.env.SetSite(r.URL.RawQuery, data)
		return

	FAIL:
		webapi.httpErrorf(w, http.StatusInternalServerError, "%s", err)

	case "DEL":
		webapi.env.DelSite(r.URL.RawQuery)

	default:
		webapi.httpError(w, http.StatusMethodNotAllowed)
	}
}

//
// Handle HTTP request
//
func (webapi *WebAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	webapi.mux.ServeHTTP(w, r)
}

//
// Return HTTP error
//
func (webapi *WebAPI) httpError(w http.ResponseWriter, status int) {
	webapi.httpErrorf(w, status, fmt.Sprintf("%s", http.StatusText(status)))
}

//
// Return HTTP error with caller-provided textual description
//
func (webapi *WebAPI) httpErrorf(w http.ResponseWriter, status int,
	format string, args ...interface{}) {

	msg := fmt.Sprintf("%d ", status)
	msg += fmt.Sprintf(format, args...)

	http.Error(w, msg, status)
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
