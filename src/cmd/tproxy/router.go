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
// Router answer
//
type RouterAnswer int

const (
	RouterBypass = RouterAnswer(iota)
	RouterForward
	RouterBlock
)

//
// RouterAnswer->string (for debugging)
//
func (a RouterAnswer) String() string {
	switch a {
	case RouterBypass:
		return "bypass"
	case RouterForward:
		return "forward"
	case RouterBlock:
		return "block"
	}

	panic("internal error")
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
func (r *Router) Route(host string) (answer RouterAnswer) {
	sites := r.env.GetSites()
	found := (*SiteParams)(nil)

	for _, site := range sites {
		if site.Host == host {
			found = &site
			break
		}

		if site.Rec &&
			strings.HasSuffix(host, site.Host) &&
			host[len(host)-len(site.Host)-1] == '.' {

			// More specific match wins
			if found == nil || len(found.Host) < len(site.Host) {
				found = &site
			}
		}
	}

	if found != nil {
		if found.Block {
			return RouterBlock
		} else {
			return RouterForward
		}
	}

	return RouterBypass
}
