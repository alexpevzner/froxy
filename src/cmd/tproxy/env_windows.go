//
// Common environment for all TProxy parts -- UNIX stuff
//

package main

/*
#define NTDDI_VERSION NTDDI_WIN7
#define INITGUID
#include <shlobj.h>
#include <knownfolders.h>

#cgo LDFLAGS: -l shell32 -l ole32

static inline void freeStr(PWSTR str) {
    CoTaskMemFree(str);
}
*/
import "C"

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

//
// Populate system-specific paths
//
func (env *Env) populateOsPaths() {
	env.PathSysConfDir = filepath.Join(getKnownFolder(&C.FOLDERID_ProgramData), "TProxy")
	env.PathUserHomeDir = getKnownFolder(&C.FOLDERID_Profile)
	env.PathUserConfDir = filepath.Join(getKnownFolder(&C.FOLDERID_LocalAppData), "TProxy")
	env.PathUserStateDir = env.PathUserConfDir
	env.PathUserLogDir = filepath.Join(env.PathUserStateDir, "log")
}

// Get known folder by FOLDERID_xxx ID
func getKnownFolder(id *C.GUID) string {
	var out C.PWSTR

	res := C.SHGetKnownFolderPath(id, 0, nil, &out)
	if res != C.S_OK {
		panic("SHGetKnownFolderPath() failed")
	}

	dir := syscall.UTF16ToString((*[1 << 16]uint16)(unsafe.Pointer(out))[:])
	C.freeStr(out)

	return dir
}

//
// Redirect stdin/stdout/stderr
//
func (env *Env) StdRedirect(stdin, stdout, stderr uintptr) error {
	C.SetStdHandle(C.STD_INPUT_HANDLE, C.HANDLE(stdin))
	C.SetStdHandle(C.STD_OUTPUT_HANDLE, C.HANDLE(stdout))
	C.SetStdHandle(C.STD_ERROR_HANDLE, C.HANDLE(stderr))

	return nil
}
