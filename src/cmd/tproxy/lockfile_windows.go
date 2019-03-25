//
// TProxy lock file -- UNIX version
//

package main

/*
#define NTDDI_VERSION NTDDI_WIN7
#include <windows.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"syscall"
)

//
// The lock file
//
type Lockfile struct {
	fd syscall.Handle // File handle
}

//
// Acquire lock file
//
func AcquireLockfile(path string) (*Lockfile, error) {
	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("%s: can't convert to UTF16")
	}

	fd, err := syscall.CreateFile(
		pathp,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0,   // Share mode
		nil, // Security Attributes
		syscall.CREATE_ALWAYS,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0, // Template file
	)

	if err == syscall.Errno(C.ERROR_SHARING_VIOLATION) {
		return nil, errors.New("TProxy already running")
	}

	if err != nil {
		return nil, err
	}

	return &Lockfile{fd}, nil
}

//
// Release lock file
//
func (l *Lockfile) Release() {
	syscall.CloseHandle(l.fd)
}
