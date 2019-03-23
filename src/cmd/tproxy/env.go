//
// Common environment for all TProxy parts
//

package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"tproxy/log"
)

//
// Environment type
//
type Env struct {
	*log.Logger
	*Ebus

	// System paths
	pathSysConfDir   string // System-wide configuration directory
	pathUserHomeDir  string // User home directory
	pathUserConfDir  string // User-specific configuration directory
	pathUserStateDir string // User specific persistent state directory

	// File paths
	pathUserConfFile  string // User-specific configuration file
	pathUserStateFile string // User-specific persistent state file
	pathUserLogFile   string // User-specific log file

	// Persistent state
	stateLock sync.RWMutex // State access lock
	state     *State       // TProxy persistent state

	// Connection state
	connStateLock sync.Mutex // Access lock
	connState     ConnState  // Current state
	connStateInfo string     // Info string

	// Statistic counters
	Counters Counters // Collection of statistic counters
}

// ----- Constructor -----
//
// Create new environment
//
func NewEnv() *Env {
	env := &Env{
		Logger: &log.DefaultLogger,
		Ebus:   NewEbus(),
		state:  &State{},
	}

	env.populateOsPaths()
	env.pathUserConfFile = filepath.Join(env.pathUserConfDir, "tproxy.cfg")
	env.pathUserStateFile = filepath.Join(env.pathUserConfDir, "tproxy.state")
	env.pathUserLogFile = filepath.Join(env.pathUserConfDir, "tproxy.log")

	os.MkdirAll(env.pathUserConfDir, 0700)
	os.MkdirAll(env.pathUserStateDir, 0700)

	err := env.state.Load(env.pathUserStateFile)
	if err != nil {
		env.state.Save(env.pathUserStateFile)
	}

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

		env.Raise(EventConnStateChanged)
	}

	env.connStateLock.Unlock()
}

// ----- Statistics counters -----
//
// Add value to the statistics counter
//
func (env *Env) AddCounter(cnt *int32, val int32) {
	atomic.AddInt32(cnt, val)
	atomic.AddUint64(&env.Counters.Tag, 1)
	env.Raise(EventCountersChanged)
}

//
// Increment the statistics counter
//
func (env *Env) IncCounter(cnt *int32) {
	env.AddCounter(cnt, 1)
}

//
// Decrement the statistics counter
//
func (env *Env) DecCounter(cnt *int32) {
	env.AddCounter(cnt, -1)
}
