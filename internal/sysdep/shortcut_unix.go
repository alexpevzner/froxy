// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// Desktop shortcuts management -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package sysdep

import (
	"fmt"
	"io/ioutil"
	"os"
)

//
// Create desktop shortcut
//
// Parameters are:
//     outpath   Output path
//     exepath   Path to executable file
//     args      Arguments
//     iconpath  Path to icon file
//     name      Program name
//     comment   Comment
//     startup   Startup or desktop shortcut
//
func CreateDesktopShortcut(
	outpath,
	exepath,
	args,
	iconpath,
	name,
	comment string,
	startup bool) error {

	// Build command with args
	cmd := exepath
	if args != "" {
		cmd += " " + args
	}

	// Create desktop entry
	text := `[Desktop Entry]
Type=Application
Version=1.0
Terminal=false`

	text += fmt.Sprintf("\nName=%s", name)
	text += fmt.Sprintf("\nComment=%s", comment)
	text += fmt.Sprintf("\nExec=%s", cmd)
	text += fmt.Sprintf("\nIcon=%s", iconpath)
	text += "\n"

	mode := os.FileMode(0755)
	if startup {
		mode = 0644
	}

	return ioutil.WriteFile(outpath, []byte(text), mode)
}
