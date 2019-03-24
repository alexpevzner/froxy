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
	"syscall"
)

//
// Install TProxy
//
func (adm *Adm) Install() error {
	return errors.New("Not implemented")
}

//
// Uninstall TProxy
//
func (adm *Adm) Uninstall() error {
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
// Kill running TProxy
//
func (adm *Adm) Kill() error {
	return errors.New("Not implemented")
}
