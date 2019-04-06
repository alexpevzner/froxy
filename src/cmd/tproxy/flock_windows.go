//
// File locking -- UNIX version
//

package main

/*
#define NTDDI_VERSION NTDDI_WIN7
#include <fileapi.h>
*/
import "C"

import (
	"os"
	"syscall"
)

//
// Lock the file
//
func FileLock(file *os.File, exclusive, wait bool) error {
	var flags C.DWORD

	if exclusive {
		flags |= C.LOCKFILE_EXCLUSIVE_LOCK
	}

	if !wait {
		flags |= C.LOCKFILE_FAIL_IMMEDIATELY
	}

	var ovp C.OVERLAPPED

	ok := C.LockFileEx(
		C.HANDLE(file.Fd()),
		flags,
		0,
		0xffffffff,
		0xffffffff,
		&ovp,
	)

	if int(ok) != 0 {
		return nil
	}

	err := syscall.GetLastError()
	if err == nil {
		err = ErrLockIsBusy
	}

	return err
}

//
// Unlock the file
//
func FileUnlock(file *os.File) error {
	var ovp C.OVERLAPPED

	ok := C.UnlockFileEx(
		C.HANDLE(file.Fd()),
		0,
		0xffffffff,
		0xffffffff,
		&ovp,
	)

	if int(ok) != 0 {
		return nil
	}

	return syscall.GetLastError()
}
