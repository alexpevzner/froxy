//
// TProxy administration (install/uninstall etc) -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"pages"
	"path/filepath"
	"syscall"
)

const desktop_entry = `[Desktop Entry]
Type=Application
Version=1.0
Name=TProxy
Terminal=false
Comment=Open TProxy configuration page in a web browser`

//
// Install TProxy
//
func (adm *Adm) Install() error {
	// Fetch icon from resources
	path := "icons/tproxy.svg"
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
	path = filepath.Join(adm.Env.PathUserConfDir, "tproxy.svg")
	err = ioutil.WriteFile(path, icon, 0644)
	if err != nil {
		return err
	}

	// Obtain name of executable file
	exec, err := os.Executable()
	if err != nil {
		return err
	}

	// Create desktop entry
	text := desktop_entry
	text += fmt.Sprintf("\nExec=%s -open", exec)
	text += fmt.Sprintf("\nIcon=%s", path)
	text += "\n"

	err = ioutil.WriteFile(adm.Env.PathUserDesktopFile, []byte(text), 0755)
	if err != nil {
		return err
	}

	// Create autostart entry
	// TODO

	return nil
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
// Open URL in a browser
//
func (adm *Adm) OpenURL(url string) error {
	return exec.Command("xdg-open", url).Start()
}
