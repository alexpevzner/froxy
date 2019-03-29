//
// TProxy lock file -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package main

import (
	"syscall"
)

//
// The lock file
//
type Lockfile struct {
	fd int // File handle
}

//
// Acquire lock file
//
func AcquireLockfile(path string) (*Lockfile, error) {
	mode := syscall.O_RDWR |
		syscall.O_CREAT |
		syscall.O_TRUNC |
		syscall.O_CLOEXEC
	fd, err := syscall.Open(path, mode, 0644)
	if err != nil {
		return nil, err
	}

	err = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return nil, ErrTProxyRunning
	}

	return &Lockfile{fd}, nil
}

//
// Release lock file
//
func (l *Lockfile) Release() {
	syscall.Close(l.fd)
}
