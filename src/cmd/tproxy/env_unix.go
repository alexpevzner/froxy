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

	env.pathSysConfDir = "/etc/tproxy"
	env.pathUserHomeDir = user.HomeDir
	env.pathUserConfDir = filepath.Join(env.pathUserHomeDir, ".tproxy")
	env.pathUserStateDir = env.pathUserConfDir
}
