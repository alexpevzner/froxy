//
// Common environment for all TProxy parts -- UNIX stuff
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package main

import (
	"fmt"
	"os"
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
}

//
// Detach stdin/stdout/stderr
//
func (env *Env) Detach() error {
	var in, out int
	var err error

	in, err = syscall.Open(os.DevNull, syscall.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("Open %q: %s", os.DevNull, err)
	}

	out, err = syscall.Open(env.PathUserLogFile,
		syscall.O_CREAT|syscall.O_WRONLY|syscall.O_APPEND, 0644)

	if err != nil {
		return fmt.Errorf("Open %q: %s", env.PathUserLogFile, err)
	}

	if err != nil {
		return fmt.Errorf("%s", err)
	}

	syscall.Dup2(in, syscall.Stdin)
	syscall.Dup2(out, syscall.Stdout)
	syscall.Dup2(out, syscall.Stderr)

	syscall.Close(in)
	syscall.Close(out)

	return nil
}
