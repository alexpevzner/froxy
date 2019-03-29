//
// TProxy administration (install/uninstall etc) -- Windows version
//

package main

/*
#define NTDDI_VERSION NTDDI_WIN7
#include <windows.h>
*/
import "C"

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

//
// Create desktop shortcut
//
func (adm *Adm) CreateDesktopShortcut(outpath, comment, args string,
	startup bool) error {

	// Lock OS thread. Otherwise OLE will get crazy
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Do the OLE mess
	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)
	oleShellObject, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer oleShellObject.Release()
	wshell, err := oleShellObject.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer wshell.Release()
	cs, err := oleutil.CallMethod(wshell, "CreateShortcut", outpath)
	if err != nil {
		return err
	}
	idispatch := cs.ToIDispatch()
	oleutil.PutProperty(idispatch, "IconLocation", adm.PathUserIconFile)
	oleutil.PutProperty(idispatch, "TargetPath", adm.OsExecutable)
	oleutil.PutProperty(idispatch, "Arguments", args)
	oleutil.PutProperty(idispatch, "Description", comment)
	oleutil.PutProperty(idispatch, "WindowStyle", 7)
	_, err = oleutil.CallMethod(idispatch, "Save")

	return err
}

//
// Create os.ProcAttr to run TProxy in background
//
func (adm *Adm) RunProcAddr() *os.ProcAttr {
	sys := &syscall.SysProcAttr{
		HideWindow: true,
		CreationFlags: uint32(C.CREATE_NO_WINDOW |
			C.DETACHED_PROCESS),
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
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
