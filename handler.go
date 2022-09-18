// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// Assortment of useful http.Handler implementations

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
	PollTag = "Froxy-Tag"
)

//
// http.Handler with poll
//
// This handler modifies handling of GET requests as follows:
// 1) For the returned data, a crypto hash of its content is
//      calculated and returned as "Froxy-Tag" header
// 2) If request contains a "Froxy-Tag" header, the handler
//      waits until calculated hash of response content becomes
//      different from hash in request
//
type HandlerWithPoll struct {
	froxy   *Froxy // Back link to Froxy
	event   Event  // Event that notifies about data change
	handler func(  // Underlying handler callback
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
		events = h.froxy.Sub(h.event)
		defer h.froxy.Unsub(events)
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
