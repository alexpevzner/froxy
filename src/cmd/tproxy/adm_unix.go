//
// TProxy administration (install/uninstall etc) -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
)

//
// Create desktop shortcut
//
func (adm *Adm) CreateDesktopShortcut(outpath, comment, args string,
	startup bool) error {
	// Obtain name of executable file
	cmd, err := os.Executable()
	if err != nil {
		return err
	}

	// Append args
	if args != "" {
		cmd += " " + args
	}

	// Create desktop entry
	text := `[Desktop Entry]
Type=Application
Version=1.0
Name=TProxy
Terminal=false`

	text += fmt.Sprintf("\nComment=%s", comment)
	text += fmt.Sprintf("\nExec=%s", cmd)
	text += fmt.Sprintf("\nIcon=%s", adm.Env.PathUserIconFile)
	text += "\n"

	mode := os.FileMode(0755)
	if startup {
		mode = 0644
	}

	return ioutil.WriteFile(outpath, []byte(text), mode)
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
