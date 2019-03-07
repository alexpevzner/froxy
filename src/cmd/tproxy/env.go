//
// Common environment for all TProxy parts
//

package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"tproxy/log"
)

//
// Environment type
//
type Env struct {
	*log.Logger

	// System paths
	pathSysConfDir   string // System-wide configuration directory
	pathUserHomeDir  string // User home directory
	pathUserConfDir  string // User-specific configuration directory
	pathUserStateDir string // User specific persistent state directory

	// File paths
	pathUserConfFile  string // User-specific configuration file
	pathUserStateFile string // User-specific persistent state file

	// Persistent state
	stateLock sync.RWMutex // State access lock
	state     *State       // TProxy persistent state
}

//
// Create new environment
//
func NewEnv() *Env {
	env := &Env{
		Logger: &log.DefaultLogger,
		state:  &State{},
	}

	env.populateOsPaths()
	env.pathUserConfFile = filepath.Join(env.pathUserConfDir, "tproxy.cfg")
	env.pathUserStateFile = filepath.Join(env.pathUserConfDir, "tproxy.state")

	os.MkdirAll(env.pathUserConfDir, 0700)
	os.MkdirAll(env.pathUserStateDir, 0700)

	env.state.Load(env.pathUserStateFile)

	return env
}

//
// Get server parameters
//
func (env *Env) GetServerParams() *StateServer {
	env.stateLock.RLock()
	s := env.state.Server
	if s == nil {
		s = &StateServer{}
	}
	env.stateLock.RUnlock()

	return s
}

//
// Set server parameters
//
func (env *Env) SetServerParams(s *StateServer) {
	env.stateLock.Lock()
	env.state.Server = s
	env.state.Save(env.pathUserStateFile)
	env.stateLock.Unlock()
}

//
// Get sites
//
func (env *Env) GetSites() (sites []StateSite) {
	env.stateLock.RLock()
	sites = env.state.Sites
	env.stateLock.RUnlock()
	return
}

//
// Set (add or update) a single site entry
//
// Please note, a site is searched by `host' parameter,
// but if site.Host != host, the existent site will be
// renamed
//
func (env *Env) SetSite(host string, site StateSite) {
	host = strings.ToLower(host)
	site.Host = strings.ToLower(site.Host)

	env.stateLock.Lock()
	defer env.stateLock.Unlock()

	// Site already listed?
	for i, s := range env.state.Sites {
		if host == s.Host {
			if s != site {
				env.state.Sites[i] = site
				goto SAVE
			}
			return // Nothing changed
		}
	}

	// New site
	env.state.Sites = append(env.state.Sites, site)

SAVE:
	env.state.Save(env.pathUserStateFile)
}

//
// Del a site
//
func (env *Env) DelSite(host string) {
	host = strings.ToLower(host)

	env.stateLock.Lock()
	defer env.stateLock.Unlock()

	// Find the site
	pos := -1
	for i, s := range env.state.Sites {
		if host == s.Host {
			pos = i
			break
		}
	}

	if pos < 0 {
		return
	}

	// Update list and save
	copy(env.state.Sites[pos:], env.state.Sites[pos+1:])
	env.state.Sites = env.state.Sites[:len(env.state.Sites)-1]

	env.state.Save(env.pathUserStateFile)
}
