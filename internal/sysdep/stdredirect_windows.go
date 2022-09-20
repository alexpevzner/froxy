// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// Redirection of stdin/stdout/stderr -- Windows version

package sysdep

/*
#define NTDDI_VERSION NTDDI_WIN7
#include <windows.h>
*/
import "C"

import (
	"os"
	"syscall"
	"unsafe"
)

//
// Redirect stdin/stdout/stderr
//
func StdRedirect(stdin, stdout, stderr uintptr) error {
	os.Stdin.Close()
	os.Stdout.Close()
	os.Stderr.Close()

	syscall.Stdin = syscall.Handle(stdin)
	syscall.Stdout = syscall.Handle(stdout)
	syscall.Stderr = syscall.Handle(stderr)

	os.Stdin = os.NewFile(uintptr(syscall.Stdin), "/dev/stdin")
	os.Stdout = os.NewFile(uintptr(syscall.Stdout), "/dev/stdout")
	os.Stderr = os.NewFile(uintptr(syscall.Stderr), "/dev/stderr")

	C.SetStdHandle(C.STD_INPUT_HANDLE, C.HANDLE(unsafe.Pointer(stdin)))
	C.SetStdHandle(C.STD_OUTPUT_HANDLE, C.HANDLE(unsafe.Pointer(stdout)))
	C.SetStdHandle(C.STD_ERROR_HANDLE, C.HANDLE(unsafe.Pointer(stderr)))

	return nil
}
