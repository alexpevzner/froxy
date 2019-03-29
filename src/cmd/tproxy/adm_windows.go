//
// TProxy administration (install/uninstall etc) -- Windows version
//

package main

/*
#define NTDDI_VERSION NTDDI_WIN7
#include <windows.h>
*/
import "C"

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

//
// Create desktop shortcut
//
func (adm *Adm) CreateDesktopShortcut(outpath, comment, args string,
	startup bool) error {
	return errors.New("Not implemented")
}

//
// Create os.ProcAttr to run TProxy in background
//
func (adm *Adm) RunProcAddr() *os.ProcAttr {
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

//
// Open URL in a browser
//
func (adm *Adm) OpenURL(url string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
