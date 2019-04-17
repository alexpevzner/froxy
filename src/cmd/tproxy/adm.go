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
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"cmd/tproxy/internal/pages"
	"cmd/tproxy/internal/sysdep"
)

//
// TProxy administration environment
//
type Adm struct {
	*Env                   // Environment
	port            int    // -p port
	OsExecutable    string // Path to executable file
	TProxyIsRunning bool   // TProxy is running
}

//
// Create new administrative environment
//
func NewAdm(env *Env, port int) (*Adm, error) {
	// ----- Create Adm structure -----
	adm := &Adm{Env: env, port: port}
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	adm.OsExecutable = exe

	// ----- Attempt to acquire tproxy.lock -----
	err = adm.TproxyLockAcquire()
	if err != nil && err != ErrTProxyRunning {
		return nil, err
	}

	adm.TProxyIsRunning = err != nil

	return adm, nil
}

//
// Install TProxy
//
func (adm *Adm) Install() error {
	// Kill TProxy if it is running
	err := adm.Kill()
	if err != nil {
		return err
	}

	// Fetch icon from resources
	_, path := filepath.Split(adm.PathUserIconFile)
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
	err = ioutil.WriteFile(adm.PathUserIconFile, icon, 0644)
	if err != nil {
		return err
	}

	// Create desktop entry
	err = sysdep.CreateDesktopShortcut(
		adm.PathUserDesktopFile,
		adm.OsExecutable,
		"-open",
		adm.PathUserIconFile,
		"Open TProxy configuration page in a web browser",
		false,
	)

	if err == nil {
		err = sysdep.CreateDesktopShortcut(
			adm.PathUserStartupFile,
			adm.OsExecutable,
			"-r",
			adm.PathUserIconFile,
			"Start TProxy service",
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
	// Kill TProxy if it is running
	err := adm.Kill()
	if err != nil {
		return err
	}

	// Remove installed files
	os.Remove(adm.PathUserDesktopFile)
	os.Remove(adm.PathUserStartupFile)
	os.Remove(adm.PathUserIconFile)

	return nil
}

//
// Run TProxy in background
//
func (adm *Adm) Run() error {
	if adm.TProxyIsRunning {
		return ErrTProxyRunning
	}

	adm.TproxyLockRelease()

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
	attr := sysdep.ProcAttrBackground()
	attr.Files = []*os.File{devnull, wstdout, wstderr}

	// Initialize process arguments
	argv := []string{
		adm.OsExecutable,
		"-p", fmt.Sprintf("%d", adm.port),
		"-detach",
	}

	// Start new process
	proc, err := os.StartProcess(adm.OsExecutable, argv, attr)
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
	// TProxy not running? Perfect, nothing to do
	if !adm.TProxyIsRunning {
		return nil
	}

	// Create shutdown request
	url := fmt.Sprintf("http://localhost:%d", adm.GetPort())
	url += "/api/shutdown"

	rq, err := http.NewRequest("TPROXY", url, nil)
	if err != nil {
		return err
	}

	// Send request and wait until connection is closed
	// Don't worry about errors too much here -- if TProxy
	// leave, we will get an error but its not a problem
	rsp, err := http.DefaultClient.Do(rq)
	if err == nil {
		io.Copy(ioutil.Discard, rsp.Body)
		rsp.Body.Close()
	}

	// And reacquire the tproxy.lock
	//
	// FIXME
	//
	// Sometimes exiting TProxy closes the connection, but
	// still continues to hold a run lock. It needs a further
	// investigation. Looks like Linux doesn't atomically release
	// resources held by an exiting process
	//
	// We will try to fix it later, but for now we have a busy-wait
	// as a temporary workaround
	for i := 0; i < 20; i++ {
		err = adm.TproxyLockAcquire()
		if err != ErrTProxyRunning {
			break
		}
		time.Sleep(time.Millisecond * 50)
	}

	if err == ErrTProxyRunning {
		err = ErrCantKillTProxy
	}
	if err == nil {
		adm.TProxyIsRunning = false
	}

	return err
}

//
// Open configuration window
//
func (adm *Adm) Open() error {
	err := adm.Run()
	if err != nil && err != ErrTProxyRunning {
		return err
	}

	url := fmt.Sprintf("http://localhost:%d", adm.GetPort())
	return sysdep.OpenURL(url)
}
