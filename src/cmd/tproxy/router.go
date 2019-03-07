//
// Request router
//

package main

import (
	"net/url"
	"path"
	"strings"
	"sync"
)

//
// Request router
//
type Router struct {
	env   *Env                // Common environment
	sites map[string]struct{} // Set of sites forwarded via server (list of glob patterns)
	lock  sync.Mutex          // Access lock
}

//
// Create new router
//
func NewRouter(env *Env) *Router {
	return &Router{
		env:   env,
		sites: make(map[string]struct{}),
	}
}

//
// Route the URL. Returns true if site must be routed via server,
// false if site must be accessed directly
//
func (r *Router) Route(url *url.URL) (forward bool) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for pattern, _ := range r.sites {
		var ok bool

		r.env.Debug("%#v %s %s", url, url.Host, url.Hostname())
		target := url.Hostname()
		if strings.IndexByte(pattern, '/') != -1 {
			target += url.Path
		}
		ok, _ = path.Match(pattern, target)

		r.env.Debug("ROUTE %s %s %v", pattern, target, ok)

		if ok {
			return true
		}
	}

	return false
}

//
// Add site to be routed via server
//
func (r *Router) AddSite(site string) {
	r.lock.Lock()
	r.sites[site] = struct{}{}
	r.lock.Unlock()
}

//
// Del site
//
func (r *Router) DelSite(site string) {
	r.lock.Lock()
	delete(r.sites, site)
	r.lock.Unlock()
}

//
// Set list of sites
//
func (r *Router) SetSites(sites []string) {
	r.lock.Lock()
	r.sites = make(map[string]struct{})
	for _, s := range sites {
		r.sites[s] = struct{}{}
	}
	r.lock.Unlock()
}
