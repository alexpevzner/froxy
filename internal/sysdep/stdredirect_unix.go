// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// Redirection of stdin/stdout/stderr -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package sysdep

import (
	"syscall"
)

//
// Redirect stdin/stdout/stderr
//
func StdRedirect(stdin, stdout, stderr uintptr) error {
	syscall.Dup2(int(stdin), syscall.Stdin)
	syscall.Dup2(int(stdout), syscall.Stdout)
	syscall.Dup2(int(stdout), syscall.Stderr)

	return nil
}
