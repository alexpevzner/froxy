//
// Desktop shortcuts management -- Windows version
//

package sysdep

/*
#define NTDDI_VERSION NTDDI_WIN7
#include <windows.h>
*/
import "C"

import (
	"runtime"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
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
	oleutil.PutProperty(idispatch, "IconLocation", iconpath)
	oleutil.PutProperty(idispatch, "TargetPath", exepath)
	oleutil.PutProperty(idispatch, "Arguments", args)
	oleutil.PutProperty(idispatch, "Description", comment)
	oleutil.PutProperty(idispatch, "WindowStyle", 7)
	_, err = oleutil.CallMethod(idispatch, "Save")

	return err
}
