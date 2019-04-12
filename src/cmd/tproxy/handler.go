//
// Assortment of useful http.Handler implementation
//

package main

import (
	"crypto/md5"
	"fmt"
	"net/http"
)

// ----- Constants -----
const (
	//
	// HTTP Header, used as a data tag for polling
	//
	PollTag = "Tproxy-Tag"
)

//
// http.Handler with poll
//
// This handler modifies handling of GET requests as follows:
// 1) For the returned data, a crypto hash of its content is
//      calculated and returned as "Tproxy-Tag" header
// 2) If request contains a "Tproxy-Tag" header, the handler
//      waits until calculated hash of response content becomes
//      different from hash in request
//
type HandlerWithPoll struct {
	tproxy  *Tproxy // Back link to Tproxy
	event   Event   // Event that notifies about data change
	handler func(   // Underlying handler callback
		w http.ResponseWriter, r *http.Request)
}

//
// Serve HTTP request
//
func (h *HandlerWithPoll) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		h.handler(w, r)
		return
	}

	// Prepare to polling for data change
	rqTag := r.Header.Get(PollTag)
	var events <-chan Event
	if rqTag != "" {
		events = h.tproxy.Sub(h.event)
		defer h.tproxy.Unsub(events)
	}

	// Serve the request
	w2 := &ResponseWriterWithBuffer{}
AGAIN:
	h.handler(w2, r)

	if w2.Status/100 == 2 {
		rspTag := fmt.Sprintf("%x", md5.Sum(w2.Bytes()))

		if events != nil && rqTag == rspTag {
			select {
			case <-events:
				w2.Reset()
				goto AGAIN
			case <-r.Context().Done():
				return
			}
		}

		w2.Header().Set(PollTag, rspTag)
	}

	w2.Send(w)
}
