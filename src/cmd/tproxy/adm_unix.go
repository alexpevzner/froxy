//
// TProxy administration (install/uninstall etc) -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package main

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
		Setsid: true,
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
