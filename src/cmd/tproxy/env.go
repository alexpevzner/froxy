//
// Common environment for all TProxy parts
//

package main

import (
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

	// Persistent state
	state *State // TProxy persistent state
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
	return env
}
