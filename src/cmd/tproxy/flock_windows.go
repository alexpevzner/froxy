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

	//
	// Note, official MSDN specification of the LockFileEx()
	// lacks information what error code is returned, when
	// LockFileEx() called with LOCKFILE_FAIL_IMMEDIATELY
	// flag, the lock is held by another process and file is
	// opened in synchronous mode
	//
	// Experimentally I've found that at this case
	// LockFileEx() returns FALSE and GetLastError()
	// returns 0
	//
	// However this blog post:
	//    https://devblogs.microsoft.com/oldnewthing/20140905-00/?p=63
	// states that LockFileEx() may return ERROR_LOCK_VIOLATION error
	// at this case
	//
	// Just in case, I check for both variants
	//
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

	return syscall.GetLastError()
}
