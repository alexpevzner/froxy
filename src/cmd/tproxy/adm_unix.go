//
// TProxy administration (install/uninstall etc) -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"unicode"
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
	// Create stdout/stderr pipes
	rstdout, wstdout, err := os.Pipe()
	if err != nil {
		return err
	}

	rstderr, wstderr, err := os.Pipe()
	if err != nil {
		return err
	}

	devnull, err := os.Open("/dev/null")
	if err != nil {
		return err
	}

	// Initialize process attributes
	sys := &syscall.SysProcAttr{
		Setsid: true,
	}
	attr := &os.ProcAttr{
		Files: []*os.File{devnull, wstdout, wstderr},
		Sys:   sys,
	}

	// Initialize process arguments
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	argv := []string{
		exe,
		"-p", fmt.Sprintf("%d", adm.Port),
		"-detach",
	}

	// Start new process
	proc, err := os.StartProcess(exe, argv, attr)
	if err != nil {
		return err
	}

	// Collect its initialization output
	wstdout.Close()
	wstderr.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	io.Copy(stdout, rstdout)
	io.Copy(stderr, rstderr)

	if stdout.Len() != 0 {
		os.Stdout.Write(stdout.Bytes())
	}

	// Check for an error
	if stderr.Len() > 0 {
		s := strings.TrimFunc(stderr.String(), unicode.IsSpace)
		proc.Kill() // Just in case
		return errors.New(s)
	}

	proc.Release()

	return nil
}

//
// Kill running TProxy
//
func (adm *Adm) Kill() error {
	return errors.New("Not implemented")
}
