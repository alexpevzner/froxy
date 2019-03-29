//
// Common environment for all TProxy parts
//

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"tproxy/log"
)

//
// Environment type
//
type Env struct {
	*log.Logger
	*Ebus

	// Directories
	PathSysConfDir   string // System-wide configuration directory
	PathUserHomeDir  string // User home directory
	PathUserConfDir  string // User-specific configuration directory
	PathUserStateDir string // User specific persistent state directory
	PathUserLogDir   string // User log dir

	// File paths
	PathUserConfFile  string // User-specific configuration file
	PathUserStateFile string // User-specific persistent state file
	PathUserLogFile   string // User-specific log file
	PathUserLockFile  string // User-specific lock file

	// Persistent state
	stateLock sync.RWMutex // State access lock
	state     *State       // TProxy persistent state

	// tproxy.lock
	tproxyLock *Lockfile // Handle of tproxy.lock

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

	// Populate paths
	env.populateOsPaths()
	env.PathUserConfFile = filepath.Join(env.PathUserConfDir, "tproxy.cfg")
	env.PathUserStateFile = filepath.Join(env.PathUserConfDir, "tproxy.state")
	env.PathUserLogFile = filepath.Join(env.PathUserLogDir, "tproxy.log")
	env.PathUserLockFile = filepath.Join(env.PathUserConfDir, "tproxy.lock")

	// Create directories
	done := make(map[string]struct{})
	for _, dir := range []string{env.PathUserConfDir, env.PathUserStateDir, env.PathUserLogDir} {
		_, ok := done[dir]
		if !ok {
			done[dir] = struct{}{}
			os.MkdirAll(dir, 0700)
		}

	}

	// Load state
	env.state.Load(env.PathUserStateFile)

	return env
}

// ----- Multiple run avoidance -----
//
// Acquire tproxy.lock
//
func (env *Env) TproxyLockAcquire() error {
	if env.tproxyLock != nil {
		panic("internal error")
	}

	// Acquire a lock file
	lock, err := AcquireLockfile(env.PathUserLockFile)
	if err != nil {
		return err
	}

	env.tproxyLock = lock
	return nil
}

//
// Release tproxy.lock
//
func (env *Env) TproxyLockRelease() {
	if env.tproxyLock == nil {
		panic("internal error")
	}

	env.tproxyLock.Release()
	env.tproxyLock = nil
}

// ----- stdin/stdout/stderr redirection -----
//
// Detach stdin/stdout/stderr from console
//
func (env *Env) Detach() error {
	nul, err := syscall.Open(os.DevNull, syscall.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("Open %q: %s", os.DevNull, err)
	}

	logfile, err := NewLogfile(env.PathUserLogFile)
	if err != nil {
		return err
	}

	out := logfile.Fd()

	return env.StdRedirect(uintptr(nul), out, out)
}

// ----- Persistent configuration -----
//
// Set TCP port
//
func (env *Env) SetPort(port int) {
	env.stateLock.Lock()
	env.state.Port = port
	env.state.Save(env.PathUserStateFile)
	env.stateLock.Unlock()
}

//
// Get TCP port
//
func (env *Env) GetPort() int {
	env.stateLock.RLock()
	port := env.state.Port
	if port == 0 {
		port = HTTP_SERVER_PORT
	}
	env.stateLock.RUnlock()

	return port
}

//
// Get server parameters
//
func (env *Env) GetServerParams() ServerParams {
	env.stateLock.RLock()
	s := env.state.Server
	env.stateLock.RUnlock()

	return s
}

//
// Set server parameters
//
func (env *Env) SetServerParams(s ServerParams) {
	env.stateLock.Lock()
	env.state.Server = s
	env.state.Save(env.PathUserStateFile)
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

	// Acquire state lock
	env.stateLock.Lock()
	defer env.stateLock.Unlock()

	// Create a copy of sites list. Router may work with
	// previous version, so instead of patching the list,
	// we will atomically replace it with the new version
	sites := make([]SiteParams, len(env.state.Sites))
	copy(sites, env.state.Sites)

	// Site already listed?
	for i, s := range sites {
		if host == s.Host {
			if s != site {
				sites[i] = site
				goto SAVE
			}
			return // Nothing changed
		}
	}

	// New site
	sites = append(sites, site)

SAVE:
	env.state.Sites = sites
	env.state.Save(env.PathUserStateFile)
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

	env.state.Save(env.PathUserStateFile)
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

// ----- Events -----
//
// Notify environment that TProxy is about to start
//
func (env *Env) Startup() {
	if env.tproxyLock == nil {
		panic("internal error")
	}

	go env.eventGoroutine()
	env.Raise(EventStartup)
}

//
// This goroutine monitors all events
//
func (env *Env) eventGoroutine() {
	events := env.Sub()
	for {
		e := <-events
		env.Debug("%s", e)
		switch e {
		case EventShutdownRequested:
			os.Exit(0)
		}
	}
}
