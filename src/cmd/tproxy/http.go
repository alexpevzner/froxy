//
// HTTP utilities
//

package main

import (
	"fmt"
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
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
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
// Fail HTTP request with formatted error message
//
func httpErrorf(w http.ResponseWriter, status int,
	format string, args ...interface{}) {
	msg := fmt.Sprintf("%d ", status)
	msg += fmt.Sprintf(format, args...)

	http.Error(w, msg, status)

}

//
// Fail HTTP request
//
func httpError(w http.ResponseWriter, status int) {
	httpErrorf(w, status, fmt.Sprintf("%s", http.StatusText(status)))
}
