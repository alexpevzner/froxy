//
// Syspem-dependent os.ProcAttr filling -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

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
