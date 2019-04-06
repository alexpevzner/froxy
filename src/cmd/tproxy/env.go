//
// Common environment for all TProxy parts
//

package main

import (
	"cmd/tproxy/internal/log"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

//
// Environment type
//
type Env struct {
	*log.Logger

	// Directories
	PathSysConfDir     string // System-wide configuration directory
	PathUserHomeDir    string // User home directory
	PathUserConfDir    string // User-specific configuration directory
	PathUserStateDir   string // User specific persistent state directory
	PathUserLogDir     string // User log dir
	PathUserKeysDir    string // User keys directory
	PathUserDesktopDir string // User Desktop folder
	PathUserStartupDir string // User Startup folder
	PathUserIconsDir   string // User icons directory
	PathUserLockDir    string // User locks

	// File paths
	PathUserConfFile    string // User-specific configuration file
	PathUserStateFile   string // User-specific persistent state file
	PathUserLogFile     string // User-specific log file
	PathUserLockFile    string // User-specific lock file
	PathUserDesktopFile string // User-specific desktop entry
	PathUserStartupFile string // User-specific startup entry
	PathUserIconFile    string // Path to icon file

	// Persistent state
	stateLock sync.RWMutex // State access lock
	state     *State       // TProxy persistent state

	// Locks
	locksCond  *sync.Cond           // To synchronize between goroutines
	locksFiles map[EnvLock]*os.File // Currently open lock files
}

//
// Global locks
//
type EnvLock string

const (
	EnvLockRun   = EnvLock("run")
	EnvLockState = EnvLock("state")
	EnvLockKeys  = EnvLock("keys")
)

// ----- Constructor -----
//
// Create new environment
//
func NewEnv() *Env {
	env := &Env{
		Logger:     &log.DefaultLogger,
		state:      &State{},
		locksCond:  sync.NewCond(&sync.Mutex{}),
		locksFiles: make(map[EnvLock]*os.File),
	}

	// Populate paths
	env.populateOsPaths()
	env.PathUserConfFile = filepath.Join(env.PathUserConfDir, "tproxy.cfg")
	env.PathUserStateFile = filepath.Join(env.PathUserConfDir, "tproxy.state")
	env.PathUserLogFile = filepath.Join(env.PathUserLogDir, "tproxy.log")
	env.PathUserLockFile = filepath.Join(env.PathUserConfDir, "tproxy.lock")

	// Create directories
	done := make(map[string]struct{})
	for _, dir := range []string{env.PathUserConfDir,
		env.PathUserStateDir,
		env.PathUserLogDir,
		env.PathUserKeysDir,
		env.PathUserLockDir} {

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

// ----- Global locks -----
//
// Acquire the global lock
//
func (env *Env) LockAcquire(lock EnvLock, wait bool) error {
	env.locksCond.L.Lock()
	defer env.locksCond.L.Unlock()

	// Synchronize with other goroutines trying to
	// acquire the same lock
	for file := env.locksFiles[lock]; file != nil; {
		if !wait {
			return ErrLockIsBusy
		}
		env.locksCond.Wait()
		file = env.locksFiles[lock]
	}

	// Create a lock file
	path := filepath.Join(env.PathUserLockDir, string(lock))
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		goto EXIT
	}

	env.locksFiles[lock] = file

	// Try to acquire a file lock
	env.locksCond.L.Unlock()
	err = FileLock(file, true, wait)
	env.locksCond.L.Lock()

EXIT:
	if err != nil {
		if file != nil {
			file.Close()
			env.locksFiles[lock] = nil
		}
		env.locksCond.Signal()
	}

	return err
}

//
// Convenience wrapper on LockAcquire() -- acquire lock with wait
//
func (env *Env) LockWait(lock EnvLock) error {
	return env.LockAcquire(lock, true)
}

//
// Convenience wrapper on LockAcquire() -- acquire lock without wait
//
func (env *Env) LockTry(lock EnvLock) error {
	return env.LockAcquire(lock, false)
}

//
// Release the global lock
//
func (env *Env) LockRelease(lock EnvLock) {
	env.locksCond.L.Lock()
	file := env.locksFiles[lock]
	if file == nil {
		panic("internal error")
	}
	file.Close()
	env.locksCond.Signal()
	env.locksCond.L.Unlock()
}

// ----- Multiple run avoidance -----
//
// Acquire tproxy.lock
//
func (env *Env) TproxyLockAcquire() error {
	err := env.LockTry(EnvLockRun)
	if err == ErrLockIsBusy {
		err = ErrTProxyRunning
	}
	return err
}

//
// Release tproxy.lock
//
func (env *Env) TproxyLockRelease() {
	env.LockRelease(EnvLockRun)
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

	// Acquire state lock
	env.stateLock.Lock()
	defer env.stateLock.Unlock()

	// Create a copy of sites list
	sites := make([]SiteParams, len(env.state.Sites))
	copy(sites, env.state.Sites)

	// Find the site
	pos := -1
	for i, s := range sites {
		if host == s.Host {
			pos = i
			break
		}
	}

	if pos < 0 {
		return
	}

	// Update list and save
	copy(sites[pos:], sites[pos+1:])
	env.state.Sites = sites[:len(sites)-1]

	env.state.Save(env.PathUserStateFile)
}
