//
// JS API for configuration pages
//

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

//
// JS API handler
//
type WebAPI struct {
	tproxy   *Tproxy                 // Back link to the Tproxy
	mux      *http.ServeMux          // Requests multiplexer
	handlers map[string]http.Handler // Table of poll-capable handlers
}

var _ = http.Handler(&WebAPI{})

//
// WebAPI handler
//
type webapiHandler struct {
	event  Event         // Event to poll for changes in GET response
	method func(*WebAPI, // WebAPI method to call
		http.ResponseWriter,
		*http.Request)
}

//
// Create new JS API handler instance
//
func NewWebAPI(tproxy *Tproxy) *WebAPI {
	webapi := &WebAPI{
		tproxy: tproxy,
		mux:    http.NewServeMux(),
	}

	webapi.handlers = map[string]http.Handler{
		"/api/server":   &HandlerWithPoll{tproxy, EventServerParamsChanged, webapi.handleServer},
		"/api/sites":    &HandlerWithPoll{tproxy, EventSitesChanged, webapi.handleSites},
		"/api/state":    &HandlerWithPoll{tproxy, EventConnStateChanged, webapi.handleState},
		"/api/counters": &HandlerWithPoll{tproxy, EventCountersChanged, webapi.handleCounters},
		"/api/keys":     &HandlerWithPoll{tproxy, EventKeysChanged, webapi.handleKeys},
	}

	// Install handlers
	for path, handler := range webapi.handlers {
		webapi.mux.Handle(path, handler)
	}

	webapi.mux.HandleFunc("/api/poll", webapi.handlePoll)
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
		conf := IDNServerParams(webapi.tproxy.GetServerParams())
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
		webapi.tproxy.Raise(EventServerParamsChanged)
		return

	FAIL:
		webapi.replyError(w, r, http.StatusInternalServerError, err)

	default:
		webapi.replyError(w, r, http.StatusMethodNotAllowed, nil)
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
			webapi.replyError(w, r,
				http.StatusInternalServerError, err)
			return
		}

		host = IDNEncode(host)
	}

	// Handle request
	switch r.Method {
	case "GET":
		conf := (IDNSiteParamsList)(webapi.tproxy.GetSites())
		webapi.replyJSON(w, conf)

	case "PUT":
		body, err := ioutil.ReadAll(r.Body)
		var data IDNSiteParams

		if err == nil {
			err = json.Unmarshal(body, &data)
		}

		if err == nil {
			webapi.tproxy.SetSite(host, SiteParams(data))
			webapi.tproxy.Raise(EventSitesChanged)
		} else {
			webapi.replyError(w, r, http.StatusInternalServerError, err)
		}

	case "DEL":
		webapi.tproxy.DelSite(host)
		webapi.tproxy.Raise(EventSitesChanged)

	default:
		webapi.replyError(w, r, http.StatusMethodNotAllowed, nil)
	}
}

//
// Handle /api/state requests
//
// GET /api/state - get connectivity state
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
		webapi.replyError(w, r, http.StatusMethodNotAllowed, nil)
		return
	}

	state, info := webapi.tproxy.GetConnState()
	stateName, stateInfo := state.Strings()
	if info == "" {
		info = stateInfo
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
			webapi.replyError(w, r,
				http.StatusInternalServerError, ErrKeyIdMissed)
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
			webapi.replyError(w, r,
				http.StatusInternalServerError, err)
			return
		}
	}

	// Handle request
	var err error
	switch r.Method {
	case "GET":
		webapi.replyJSON(w, webapi.tproxy.GetKeys())
		return

	case "PUT":
		err = webapi.tproxy.KeyMod(id, &info)

	case "DEL":
		err = webapi.tproxy.KeyDel(id)

	case "POST":
		_, err = webapi.tproxy.KeyGen(&info)
	}

	if err != nil {
		webapi.replyError(w, r, http.StatusInternalServerError, err)
	} else {
		webapi.tproxy.Raise(EventKeysChanged)
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
		webapi.replyError(w, r, http.StatusMethodNotAllowed, nil)
		return
	}

	webapi.replyJSON(w, &webapi.tproxy.Counters)
}

//
// Handle /api/poll requests
//
// Implements Websocket-based poll for changes
//
func (webapi *WebAPI) handlePoll(w http.ResponseWriter, r *http.Request) {
	upgrader := &websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		webapi.tproxy.Debug("poll %s", err)
		webapi.replyError(w, r, http.StatusMethodNotAllowed, nil)
		return
	}

	// Websocket message
	type msg struct {
		Path string          `json:"path"`           // Request path (i.e., "/api/counters"
		Tag  string          `json:"tag,omitempty"`  // Data tag
		Data json.RawMessage `json:"data,omitempty"` // Data (in responses only)
	}

	// Reader goroutine
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		defer conn.Close()

		writeLock := &sync.Mutex{}
		for {
			// Receive next message
			var in msg
			err := conn.ReadJSON(&in)
			if err != nil {
				return
			}

			// Perform a request in background
			go func() {
				// Initialize response message
				out := msg{
					Path: in.Path,
				}

				// Execute a request
				if h := webapi.handlers[in.Path]; h != nil {
					r, _ = http.NewRequest("GET", in.Path, nil)
					r.Header.Set(PollTag, in.Tag)
					r = r.WithContext(ctx)
					w := &ResponseWriterWithBuffer{}

					h.ServeHTTP(w, r)

					if w.Status/100 == 2 {
						out.Tag = w.Header().Get(PollTag)
						out.Data = json.RawMessage(w.Bytes())
					}
				}

				// Send a response
				writeLock.Lock()
				conn.WriteJSON(out)
				writeLock.Unlock()
			}()
		}
	}()
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
		webapi.replyError(w, r, http.StatusMethodNotAllowed, nil)
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
// For the GET requests it automatically implements a long polling
// for data change:
//   1) For the returned data, a crypto hash of its content is
//      calculated and returned as "Tproxy-Tag" header
//   2) If request contains a "Tproxy-Tag" header, the handler
//      waits until calculated hash of response content becomes
//      different from hash in request
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
	httpNoCache(w)

	w.Write(body)
	w.Write([]byte("\n"))
}

//
// Reply with HTTP error
//
func (webapi *WebAPI) replyError(w http.ResponseWriter, r *http.Request,
	status int, err error) {

	// FIXME - send JSON error object instead
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	httpNoCache(w)

	w.WriteHeader(status)

	if err != nil {
		fmt.Fprintf(w, "%s\n", err)
	} else {
		switch status {
		case http.StatusMethodNotAllowed:
			fmt.Fprintf(w, "Method %q not allowed\n", r.Method)
		default:
			fmt.Fprintf(w, "%s\n", http.StatusText(status))
		}
	}
}
