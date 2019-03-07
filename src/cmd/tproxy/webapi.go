//
// JS API for configuration pages
//

package main

import (
	"net/http"
)

//
// JS API handler
//
type WebAPI struct {
	env *Env
}

var _ = http.Handler(&WebAPI{})

//
// Create new JS API handler instance
//
func NewWebAPI(env *Env) *WebAPI {
	return &WebAPI{
		env: env,
	}
}

//
// Handle HTTP request
//
func (webapi *WebAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "TODO", http.StatusNotImplemented)
}
