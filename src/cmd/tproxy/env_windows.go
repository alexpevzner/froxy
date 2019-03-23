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
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

//
// Populate system-specific paths
//
func (env *Env) populateOsPaths() {
	progdata := getKnownFolder(&C.FOLDERID_ProgramData)
	env.PathSysConfDir = filepath.Join(progdata, "TProxy")

	env.PathUserHomeDir = getKnownFolder(&C.FOLDERID_Profile)
	env.PathUserConfDir = getKnownFolder(&C.FOLDERID_LocalAppData)
	env.PathUserStateDir = env.PathUserConfDir
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
// Detach stdin/stdout/stderr
//
func (env *Env) Detach() error {
	nul, err := syscall.Open(os.DevNull, syscall.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("Open %q: %s", os.DevNull, err)
	}

	log, err := syscall.Open(env.PathUserLogFile,
		syscall.O_CREAT|syscall.O_WRONLY|syscall.O_APPEND, 0644)

	if err != nil {
		return fmt.Errorf("Open %q: %s", env.PathUserLogFile, err)
	}

	C.SetStdHandle(C.STD_INPUT_HANDLE, C.HANDLE(nul))
	C.SetStdHandle(C.STD_OUTPUT_HANDLE, C.HANDLE(log))
	C.SetStdHandle(C.STD_ERROR_HANDLE, C.HANDLE(log))

	return nil
}
