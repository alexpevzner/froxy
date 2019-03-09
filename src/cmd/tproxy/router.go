//
// Request router
//

package main

import (
	"strings"
)

//
// Request router
//
type Router struct {
	env *Env // Common environment
}

//
// Create new router
//
func NewRouter(env *Env) *Router {
	return &Router{
		env: env,
	}
}

//
// Route the URL. Returns true if site must be routed via server,
// false if site must be accessed directly
//
func (r *Router) Route(host string) (forward bool) {
	sites := r.env.GetSites()
	for _, site := range sites {
		r.env.Debug("%s vs %s", host, site.Host)

		if site.Host == host {
			return true
		}

		r.env.Debug("has suffix: %v", strings.HasSuffix(host, site.Host))

		if site.Rec &&
			strings.HasSuffix(host, site.Host) &&
			host[len(host)-len(site.Host)-1] == '.' {
			return true
		}
	}

	return false
}
