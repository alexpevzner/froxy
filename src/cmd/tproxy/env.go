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

	// Connection state
	connStateLock          sync.Mutex    // Access lock
	connState              ConnState     // Current state
	connStateInfo          string        // Info string
	connStateChan          chan struct{} // Wait chanell
	connStateChanSignalled bool
}

// ----- Constructor -----
//
// Create new environment
//
func NewEnv() *Env {
	env := &Env{
		Logger:        &log.DefaultLogger,
		state:         &State{},
		connStateChan: make(chan struct{}),
	}

	env.populateOsPaths()
	env.pathUserConfFile = filepath.Join(env.pathUserConfDir, "tproxy.cfg")
	env.pathUserStateFile = filepath.Join(env.pathUserConfDir, "tproxy.state")

	os.MkdirAll(env.pathUserConfDir, 0700)
	os.MkdirAll(env.pathUserStateDir, 0700)

	env.state.Load(env.pathUserStateFile)

	return env
}

// ----- Persistent configuration -----
//
// Get server parameters
//
func (env *Env) GetServerParams() *ServerParams {
	env.stateLock.RLock()
	s := env.state.Server
	if s == nil {
		s = &ServerParams{}
	}
	env.stateLock.RUnlock()

	return s
}

//
// Set server parameters
//
func (env *Env) SetServerParams(s *ServerParams) {
	env.stateLock.Lock()
	env.state.Server = s
	env.state.Save(env.pathUserStateFile)
	env.stateLock.Unlock()
}

//
// Get sites
//
func (env *Env) GetSites() (sites []SiteParams) {
	env.stateLock.RLock()
	sites = env.state.Sites
	if sites == nil {
		sites = make([]SiteParams, 0)
	}
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
func (env *Env) SetSite(host string, site SiteParams) {
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

// ----- Connection state -----
//
// Connection states
//
type ConnState int

const (
	ConnNotConfigured = ConnState(iota)
	ConnTrying
	ConnEstablished
)

//
// ConnState -> ("name", "default info string")
//
func (s ConnState) Strings() (string, string) {
	switch s {
	case ConnNotConfigured:
		return "noconfig", "Server not configured"
	case ConnTrying:
		return "trying", ""
	case ConnEstablished:
		return "established", "Connected to the server"
	}

	panic("internal error")
}

//
// Get connection state
//
func (env *Env) GetConnState() (state ConnState, info string) {
	env.connStateLock.Lock()
	state = env.connState
	info = env.connStateInfo
	env.connStateLock.Unlock()

	return
}

//
// Set connection state
//
func (env *Env) SetConnState(state ConnState, info string) {
	env.connStateLock.Lock()

	if env.connState != state {
		env.connState = state
		env.connStateInfo = info

		if !env.connStateChanSignalled {
			env.connStateChanSignalled = true
			close(env.connStateChan)
		}
	}

	env.connStateLock.Unlock()
}

//
// Get wait channel for connection state change
//
// The channel becomes "readable" when state changes
//
func (env *Env) ConnStateChan() (c <-chan struct{}) {
	env.connStateLock.Lock()

	c = env.connStateChan
	if env.connStateChanSignalled {
		env.connStateChanSignalled = false
		env.connStateChan = make(chan struct{})
	}

	env.connStateLock.Unlock()

	return c
}
