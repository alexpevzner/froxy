// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// Syspem-dependent os.ProcAttr filling -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package sysdep

import (
	"os"
	"syscall"
)

//
// Create os.ProcAttr to run process in background
//
func ProcAttrBackground() *os.ProcAttr {
	sys := &syscall.SysProcAttr{
		Setsid: true,
	}
	attr := &os.ProcAttr{
		Sys: sys,
	}
	return attr
}
