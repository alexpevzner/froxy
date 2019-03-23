//
// Common environment for all TProxy parts -- UNIX stuff
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package main

import (
	"os/user"
	"path/filepath"
)

//
// Populate system-specific paths
//
func (env *Env) populateOsPaths() {
	user, err := user.Current()
	if err != nil {
		panic(err.Error())
	}

	env.PathSysConfDir = "/etc/tproxy"
	env.PathUserHomeDir = user.HomeDir
	env.PathUserConfDir = filepath.Join(env.PathUserHomeDir, ".tproxy")
	env.PathUserStateDir = env.PathUserConfDir
}
