//
// Open URL in a browser -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package sysdep

import (
	"os/exec"
)

//
// Open URL in a browser
//
func OpenURL(url string) error {
	return exec.Command("xdg-open", url).Start()
}
