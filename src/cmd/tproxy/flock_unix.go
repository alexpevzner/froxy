//
// File locking -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package main

import (
	"os"
	"syscall"
)

//
// Lock the file
//
func FileLock(file *os.File, exclusive, wait bool) error {
	var how int

	if exclusive {
		how = syscall.LOCK_EX
	} else {
		how = syscall.LOCK_SH
	}

	if !wait {
		how |= syscall.LOCK_NB
	}

	return syscall.Flock(int(file.Fd()), how)
}

//
// Unlock the file
//
func FileUnlock(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
