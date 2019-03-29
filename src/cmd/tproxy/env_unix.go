//
// Common environment for all TProxy parts -- UNIX stuff
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package main

import (
	"os/user"
	"path/filepath"
	"syscall"
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
	env.PathUserLogDir = filepath.Join(env.PathUserStateDir, "log")
	env.PathUserDesktopDir = filepath.Join(env.PathUserHomeDir, "Desktop")
	env.PathUserStartupDir = filepath.Join(env.PathUserHomeDir, ".config/autostart")
	env.PathUserIconsDir = env.PathUserConfDir

	env.PathUserDesktopFile = filepath.Join(env.PathUserDesktopDir, "tproxy.desktop")
	env.PathUserStartupFile = filepath.Join(env.PathUserStartupDir, "tproxy.desktop")
	env.PathUserIconFile = filepath.Join(env.PathUserIconsDir, "tproxy.svg")
}

//
// Redirect stdin/stdout/stderr
//
func (env *Env) StdRedirect(stdin, stdout, stderr uintptr) error {
	syscall.Dup2(int(stdin), syscall.Stdin)
	syscall.Dup2(int(stdout), syscall.Stdout)
	syscall.Dup2(int(stdout), syscall.Stderr)

	return nil
}
