//
// File locking -- UNIX version
//

package main

/*
#define NTDDI_VERSION NTDDI_WIN7
#include <fileapi.h>
#include <windows.h>
*/
import "C"

import (
	"os"
	"runtime"
	"syscall"
)

//
// Lock the file
//
func FileLock(file *os.File, exclusive, wait bool) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

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

	switch errno := C.GetLastError(); errno {
	case C.NO_ERROR, C.ERROR_LOCK_VIOLATION:
		return ErrLockIsBusy
	default:
		return syscall.Errno(errno)
	}
}

//
// Unlock the file
//
func FileUnlock(file *os.File) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

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

	switch errno := C.GetLastError(); errno {
	case C.NO_ERROR:
		return nil
	default:
		return syscall.Errno(errno)
	}
}
