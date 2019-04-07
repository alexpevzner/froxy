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
	webapi.mux.HandleFunc("/api/keys", webapi.handleKeys)
	webapi.mux.HandleFunc("/api/shutdown", webapi.handleShutdown)

	return webapi
}

//
// Handle /api/server requests
//
// GET - get server parameters. Returns IDNServerParams structure
// PUT - set server parameters. Receives IDNServerParams structure
//
func (webapi *WebAPI) handleServer(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		conf := struct {
			// Server parameters
			IDNServerParams

			// Some additional info to simplify life of the
			// Configuration page javascript
			HasKeys bool `json:"haskeys"` // User has SSH keys
		}{
			IDNServerParams: (IDNServerParams)(webapi.tproxy.GetServerParams()),
			HasKeys:         webapi.tproxy.HasKeys(),
		}
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
		httpError(w, http.StatusInternalServerError, err)

	default:
		httpError(w, http.StatusMethodNotAllowed, nil)
	}
}

//
// Handle /api/sites requests
//
// GET /api/sites      - get all sites
// DEL /api/sites?host - del particular site
// PUT /api/sites?host - set particular site.
//                       Receives IDNSiteParams structure
//
// Note, PUT identifies site by query parameter, not by the Host
// field of the IDNSiteParams structure. So to change host name
// in the existing record, query parameter must point to the
// existent host, and structure must contain a new host name
//
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
			httpError(w, http.StatusInternalServerError, err)
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
		httpError(w, http.StatusInternalServerError, err)

	case "DEL":
		webapi.tproxy.DelSite(host)

	default:
		httpError(w, http.StatusMethodNotAllowed, nil)
	}
}

//
// Handle /api/state requests
//
// GET /api/state[?prev] - get connectivity state
//
// Returns the following JSON object:
//     {
//         "state": "noconfig" | "trying" | "established",
//         "info":  "some human-readable explanation"
//     }
//
// If query parameter present, GET waits until state becomes
// different from the previous state, as set in the query
//
func (webapi *WebAPI) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		httpError(w, http.StatusMethodNotAllowed, nil)
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
// Handle /api/keys requests
//
// GET /api/keys    - get list of keys as array of KeyInfo structures
// PUT /api/keys?id - update existent key. Accepts KeyInfo structure,
//                    but only Comment and Enabled parameters can be
//                    updated (other fields are ignored)
// DEL /api/keys?id - Deletes a key
// POST /api/keys   - Accepts KeyInfo structure and generates new
//                    key. Fingerprints are ignored, if present
//
// Id parameter, where required, is the sha-256 fingerprint of
// the key
//
func (webapi *WebAPI) handleKeys(w http.ResponseWriter, r *http.Request) {
	// Obtain key id, if required
	var id string
	if r.Method == "PUT" || r.Method == "DEL" {
		id = r.URL.RawQuery
		if id == "" {
			httpError(w, http.StatusInternalServerError, ErrKeyIdMissed)
			return
		}
	}

	// Decode KeyInfo structure, if required
	var info KeyInfo
	if r.Method == "PUT" || r.Method == "POST" {
		body, err := ioutil.ReadAll(r.Body)

		if err == nil {
			err = json.Unmarshal(body, &info)
		}

		if err != nil {
			httpError(w, http.StatusInternalServerError, err)
			return
		}
	}

	// Handle request
	var err error
	switch r.Method {
	case "GET":
		webapi.replyJSON(w, webapi.tproxy.GetKeys())

	case "PUT":
		err = webapi.tproxy.KeyMod(id, &info)

	case "DEL":
		err = webapi.tproxy.KeyDel(id)

	case "POST":
		webapi.tproxy.Debug("rq=%#v", info)
		_, err = webapi.tproxy.KeyGen(&info)
	}

	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
	}
}

//
// Handle /api/counters requests
//
// GET /api/counters[?tag] - returns Counters structure
//
// If query parameter present, GET waits until Counters.Tag
// becomes different from the tag, specified in the query.
// Note, Counters.Tag increments at every Counters update
//
func (webapi *WebAPI) handleCounters(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		httpError(w, http.StatusMethodNotAllowed, nil)
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
// TPROXY /api/shudtown - initiates TProxy shutdown. Connection
//                        remains open until TProxy process termination,
//                        so requester may synchronize with shutdown
//                        completion
//
func (webapi *WebAPI) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != "TPROXY" {
		httpError(w, http.StatusMethodNotAllowed, nil)
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
