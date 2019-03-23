//
// TProxy administration (install/uninstall etc) -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package main

import (
	"errors"
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
// Run TProxy in background
//
func (adm *Adm) Run() error {
	return errors.New("Not implemented")
}

//
// Kill running TProxy
//
func (adm *Adm) Kill() error {
	return errors.New("Not implemented")
}
