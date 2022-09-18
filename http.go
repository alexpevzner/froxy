// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// HTTP utilities

package main

import (
	"net/http"
	"net/textproto"
	"strings"
)

//
// List of HTTP hop-by-hop headers
//
var httpHopByHopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Connection",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
}

//
// Copy HTTP headers
//
func httpCopyHeaders(dst, src http.Header) {
	for k, v := range src {
		dst[k] = v
	}
}

//
// Remove hop-by-hop headers.
//
// Upgrade headers are preserved, and if present, this function
// returns true
//
func httpRemoveHopByHopHeaders(hdr http.Header) bool {
	// We must delete headers listed in Connection
	if c, ok := hdr["Connection"]; ok {
		for _, v := range c {
			for _, k := range strings.Split(v, ",") {
				if k = strings.TrimSpace(k); k != "" {
					k = textproto.CanonicalMIMEHeaderKey(k)
					if k != "Upgrade" {
						delete(hdr, k)
					}
				}
			}
		}
	}

	// And also standard Hop-by-hop headers
	for _, k := range httpHopByHopHeaders {
		delete(hdr, k)
	}

	// Restore "Connection: Upgrade" header
	_, upgraded := hdr["Upgrade"]
	if upgraded {
		hdr["Connection"] = []string{"Upgrade"}
	}

	return upgraded
}

//
// Set response headers to disable cacheing
//
func httpNoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}
