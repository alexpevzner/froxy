//
// Desktop shortcuts management -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package sysdep

import (
	"fmt"
	"io/ioutil"
	"os"
)

//
// Create desktop shortcut
//
func CreateDesktopShortcut(
	outpath,
	exepath,
	args,
	iconpath,
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
Name=TProxy
Terminal=false`

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
