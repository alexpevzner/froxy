// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// System-dependent paths -- Windows stuff

package sysdep

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
// Get system configuration directory for the program
//
func SysConfDir(program string) string {
	return filepath.Join(getKnownFolder(&C.FOLDERID_ProgramData), program)
}

//
// Get user home directory
//
func UserHomeDir() string {
	return getKnownFolder(&C.FOLDERID_Profile)
}

//
// Get user configuration directory for the program
//
func UserConfDir(program string) string {
	return filepath.Join(getKnownFolder(&C.FOLDERID_LocalAppData), program)
}

//
// Get user desktop directory
//
func UserDesktopDir() string {
	return getKnownFolder(&C.FOLDERID_Desktop)
}

//
// Get user startup (autostart) directory
//
func UserStartupDir() string {
	return getKnownFolder(&C.FOLDERID_Startup)
}

//
// Get user desktop file for the program
//
func UserDesktopFile(program, iconname string) string {
	return filepath.Join(UserDesktopDir(), iconname+".lnk")
}

//
// Get user startup file for the program
//
func UserStartupFile(program, iconname string) string {
	return filepath.Join(UserStartupDir(), iconname+".lnk")
}

//
// Get icon file extension
//
func IconExt() string {
	return "ico"
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
