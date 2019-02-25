//
// HTTP utilities
//

package main

import (
	"net/http"
)

//
// List of HTTP hop-by-hop headers
//
var httpHopByHopHeaders = map[string]struct{}{
	"Connection":         struct{}{},
	"Keep-Alive":         struct{}{},
	"Public":             struct{}{},
	"Proxy-Authenticate": struct{}{},
	"Transfer-Encoding":  struct{}{},
	"Upgrade":            struct{}{},
	"Proxy-Connection":   struct{}{},
}

//
// Copy HTTP headers
//
func httpCopyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

//
// Remove hop-by-hop headers
//
func httpRemoveHopByHopHeaders(hdr http.Header) {
	for k, _ := range httpHopByHopHeaders {
		delete(hdr, k)
	}
}
