// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// Syspem-dependent os.ProcAttr filling -- Windows version

package sysdep

/*
#define NTDDI_VERSION NTDDI_WIN7
#include <windows.h>
*/
import "C"

import (
	"os"
	"syscall"
)

//
// Create os.ProcAttr to run process in background
//
func ProcAttrBackground() *os.ProcAttr {
	sys := &syscall.SysProcAttr{
		HideWindow: true,
		CreationFlags: uint32(C.CREATE_NO_WINDOW |
			C.DETACHED_PROCESS),
	}
	attr := &os.ProcAttr{
		Sys: sys,
	}
	return attr
}
