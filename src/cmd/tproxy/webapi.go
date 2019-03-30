//
// JS API for configuration pages
//

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

//
// JS API handler
//
type WebAPI struct {
	tproxy *Tproxy
	mux    *http.ServeMux
}

var _ = http.Handler(&WebAPI{})

//
// Create new JS API handler instance
//
func NewWebAPI(tproxy *Tproxy) *WebAPI {
	webapi := &WebAPI{
		tproxy: tproxy,
		mux:    http.NewServeMux(),
	}

	webapi.mux.HandleFunc("/api/server", webapi.handleServer)
	webapi.mux.HandleFunc("/api/sites", webapi.handleSites)
	webapi.mux.HandleFunc("/api/state", webapi.handleState)
	webapi.mux.HandleFunc("/api/counters", webapi.handleCounters)
	webapi.mux.HandleFunc("/api/shutdown", webapi.handleShutdown)

	return webapi
}

//
// Handle /api/server requests
//
func (webapi *WebAPI) handleServer(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		conf := (IDNServerParams)(webapi.tproxy.GetServerParams())
		webapi.replyJSON(w, conf)

	case "PUT":
		body, err := ioutil.ReadAll(r.Body)
		var data IDNServerParams

		if err != nil {
			goto FAIL
		}

		err = json.Unmarshal(body, &data)
		if err != nil {
			goto FAIL
		}

		webapi.tproxy.SetServerParams((ServerParams)(data))
		return

	FAIL:
		httpErrorf(w, http.StatusInternalServerError, "%s", err)

	default:
		httpError(w, http.StatusMethodNotAllowed)
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
			err = ErrHttpHostMissed
		}

		if err != nil {
			httpErrorf(w, http.StatusInternalServerError, "%s", err)
			return
		}

		webapi.tproxy.Debug("host=%s", host)
		host = IDNEncode(host)
		webapi.tproxy.Debug("host decoded=%s", host)
	}

	// Handle request
	switch r.Method {
	case "GET":
		conf := (IDNSiteParamsList)(webapi.tproxy.GetSites())
		webapi.replyJSON(w, conf)

	case "PUT":
		body, err := ioutil.ReadAll(r.Body)
		var data IDNSiteParams

		if err != nil {
			goto FAIL
		}

		err = json.Unmarshal(body, &data)
		if err != nil {
			goto FAIL
		}

		webapi.tproxy.SetSite(host, SiteParams(data))
		return

	FAIL:
		httpErrorf(w, http.StatusInternalServerError, "%s", err)

	case "DEL":
		webapi.tproxy.DelSite(host)

	default:
		httpError(w, http.StatusMethodNotAllowed)
	}
}

//
// Handle /api/state requests
//
func (webapi *WebAPI) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		httpError(w, http.StatusMethodNotAllowed)
		return
	}

	events := webapi.tproxy.Sub(EventConnStateChanged)
	defer webapi.tproxy.Unsub(events)

AGAIN:
	state, info := webapi.tproxy.GetConnState()
	stateName, stateInfo := state.Strings()
	if info == "" {
		info = stateInfo
	}

	if stateName == r.URL.RawQuery {
		select {
		case <-events:
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
		httpError(w, http.StatusMethodNotAllowed)
		return
	}

	events := webapi.tproxy.Sub(EventCountersChanged)
	defer webapi.tproxy.Unsub(events)

AGAIN:
	counters := webapi.tproxy.Counters
	tag := fmt.Sprintf("%d", counters.Tag)

	if tag == r.URL.RawQuery {
		select {
		case <-events:
			goto AGAIN
		case <-r.Context().Done():
			return
		}
	}

	webapi.replyJSON(w, &webapi.tproxy.Counters)
}

//
// Handle /api/shudtown requests
//
func (webapi *WebAPI) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != "TPROXY" {
		httpError(w, http.StatusMethodNotAllowed)
		return
	}

	// Hijack the connection so it will be closed only
	// after TProxy exit
	//
	// Here we manually generate a minimalist valid HTTP response
	// header and keep connection open until OS closes it after
	// TProxy exit. We use HTTP version 1.0, because it allows
	// (unlike HTTP 1.1) data streaming with EOF indicated
	// by closing the connection.
	//
	// When connection finally closed, client will know that
	// TProxy shutdown completed
	hijacker, ok := w.(http.Hijacker)
	if ok {
		c, _, err := hijacker.Hijack()
		if err == nil {
			c.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
		}
	}

	// Raise shutdown event
	webapi.tproxy.Raise(EventShutdownRequested)
}

//
// Handle HTTP request
//
func (webapi *WebAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	webapi.mux.ServeHTTP(w, r)
}

//
// Reply with JSON data
//
func (webapi *WebAPI) replyJSON(w http.ResponseWriter, data interface{}) {
	body, err := json.Marshal(struct {
		Data interface{} `json:"data"`
	}{data})

	if err != nil {
		panic("internal error: " + err.Error())
	}

	w.Header().Set("Content-Type", "application/json")

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	w.Write(body)
	w.Write([]byte("\n"))
}
