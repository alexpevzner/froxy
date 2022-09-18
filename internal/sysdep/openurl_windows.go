//
// Open URL in a browser -- Windows version
//

package sysdep

import (
	"os/exec"
)

//
// Open URL in a browser
//
func OpenURL(url string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
