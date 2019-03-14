//
// JS API for configuration pages
//

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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
	webapi.mux.HandleFunc("/api/state", webapi.handleState)
	webapi.mux.HandleFunc("/api/counters", webapi.handleCounters)

	return webapi
}

//
// Handle /api/server requests
//
func (webapi *WebAPI) handleServer(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		conf := webapi.env.GetServerParams()
		webapi.replyJSON(w, conf)

	case "PUT":
		body, err := ioutil.ReadAll(r.Body)
		var data ServerParams

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
// Handle /api/sites requests
//
func (webapi *WebAPI) handleSites(w http.ResponseWriter, r *http.Request) {
	var host string

	// Decode host, if required (for PUT and DEL requests)
	if r.Method == "PUT" || r.Method == "DEL" {
		var err error
		host, err = url.QueryUnescape(r.URL.RawQuery)

		if err == nil && r.Method == "DEL" && host == "" {
			err = errors.New("invalid query: host parameter missed")
		}

		if err != nil {
			webapi.httpErrorf(w, http.StatusInternalServerError, "%s", err)
			return
		}

	}

	// Handle request
	switch r.Method {
	case "GET":
		conf := webapi.env.GetSites()
		webapi.replyJSON(w, conf)

	case "PUT":
		body, err := ioutil.ReadAll(r.Body)
		var data SiteParams

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
// Handle /api/sites requests
//
func (webapi *WebAPI) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		webapi.httpError(w, http.StatusMethodNotAllowed)
		return
	}

AGAIN:
	state, info := webapi.env.GetConnState()
	stateName, stateInfo := state.Strings()
	if info == "" {
		info = stateInfo
	}

	if stateName == r.URL.RawQuery {
		select {
		case <-webapi.env.ConnStateChan():
			goto AGAIN
		case <-r.Context().Done():
			return
		}
	}

	data := struct {
		State string `json:"state"`
		Info  string `json:"info"`
	}{stateName, info}

	webapi.replyJSON(w, &data)
}

//
// Handle /api/counters requests
//
func (webapi *WebAPI) handleCounters(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		webapi.httpError(w, http.StatusMethodNotAllowed)
		return
	}

	webapi.replyJSON(w, &webapi.env.Counters)
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
	body, _ := json.Marshal(struct {
		Data interface{} `json:"data"`
	}{data})

	w.Header().Set("Content-Type", "application/json")

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	w.Write(body)
	w.Write([]byte("\n"))
}
