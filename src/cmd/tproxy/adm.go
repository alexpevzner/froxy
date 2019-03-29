//
// TProxy administration (install/uninstall etc)
//

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"pages"
	"path/filepath"
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
// Install TProxy
//
func (adm *Adm) Install() error {
	// Fetch icon from resources
	_, path := filepath.Split(adm.Env.PathUserIconFile)
	path = "icons/" + path

	var icon []byte
	iconfile, err := pages.AssetFS.Open(path)
	if err == nil {
		icon, err = ioutil.ReadAll(iconfile)
		iconfile.Close()
	}
	if err != nil {
		return fmt.Errorf("Resource %q: %s", path, err)
	}

	// Save icon to disk
	err = ioutil.WriteFile(adm.Env.PathUserIconFile, icon, 0644)
	if err != nil {
		return err
	}

	// Create desktop entry
	err = adm.CreateDesktopShortcut(
		adm.Env.PathUserDesktopFile,
		"Open TProxy configuration page in a web browser",
		"-open",
		false,
	)

	if err == nil {
		err = adm.CreateDesktopShortcut(
			adm.Env.PathUserStartupFile,
			"Start TProxy service",
			"-r",
			true,
		)
	}

	// Undo changes in a case of errors
	if err != nil {
		adm.Uninstall()
	}

	return err
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
