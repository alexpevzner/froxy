//
// TProxy administration (install/uninstall etc)
//

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"unicode"
)

//
// TProxy administration environment
//
type Adm struct {
	Port int  // -p port
	Env  *Env // Environment
}

//
// Uninstall TProxy
//
func (adm *Adm) Uninstall() error {
	os.Remove(adm.Env.PathUserDesktopFile)
	os.Remove(adm.Env.PathUserStartupFile)
	os.Remove(adm.Env.PathUserIconFile)

	return nil
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

	devnull, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}

	// Initialize process attributes
	attr := adm.RunProcAddr()
	attr.Files = []*os.File{devnull, wstdout, wstderr}

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
	url := fmt.Sprintf("http://localhost:%d", adm.Env.GetPort())
	url += "/api/shutdown"

	rq, err := http.NewRequest("TPROXY", url, nil)
	if err == nil {
		_, err = http.DefaultClient.Do(rq)
	}

	return err
}

//
// Open configuration window
//
func (adm *Adm) Open() error {
	url := fmt.Sprintf("http://localhost:%d", adm.Env.GetPort())
	return adm.OpenURL(url)
}
