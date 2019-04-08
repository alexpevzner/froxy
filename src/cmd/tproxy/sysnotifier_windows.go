//
// System events notifier -- Windows version
//

package main

/*
#define NTDDI_VERSION NTDDI_WIN7
#include <windows.h>

#cgo LDFLAGS: -l user32
*/
import "C"

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"unsafe"
)

//
// System events notifier
//
type SysNotifier struct {
	tproxy *Tproxy // Back link to Tproxy
}

//
// Create new SysNotifier
//
func NewSysNotifier(tproxy *Tproxy) *SysNotifier {
	sn := &SysNotifier{tproxy: tproxy}
	go sn.conGoroutine()
	go sn.winGoroutine()
	return sn
}

//
// This goroutine waits for console events, using Go's os/signal wrapper
//
func (sn *SysNotifier) conGoroutine() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
	sn.tproxy.Raise(EventShutdownRequested)
}

//
// This goroutine creates invisible window and waits for system messages
//
func (sn *SysNotifier) winGoroutine() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Prepare things
	name := C.CString("TProxy")
	hInstance := C.GetModuleHandle(nil)
	wndProc := C.WNDPROC(unsafe.Pointer(syscall.NewCallback(sn.wndProc)))

	// Register window class
	wndclass := C.WNDCLASSA{
		style:         0,
		lpfnWndProc:   wndProc,
		hInstance:     hInstance,
		hIcon:         nil,
		hCursor:       nil,
		hbrBackground: C.HBRUSH(unsafe.Pointer(uintptr(C.COLOR_BACKGROUND))),
		lpszMenuName:  nil,
		lpszClassName: name,
	}

	C.RegisterClassA(&wndclass)

	// Create invisible window for notifications
	C.CreateWindowExA(
		0,
		name,
		name,
		0,
		0, 0,
		400, 400,
		nil,
		nil,
		hInstance,
		nil,
	)

	// Run message loop
	for {
		var msg C.MSG
		if C.GetMessage(&msg, nil, 0, 0) == 0 {
			break
		}
		C.TranslateMessage(&msg)
		C.DispatchMessage(&msg)
	}
}

//
// SysNotifier window procedure
//
func (sn *SysNotifier) wndProc(hWnd C.HWND, msg C.UINT, wParam C.WPARAM, lParam C.LPARAM) C.LRESULT {
	sn.tproxy.Debug("msg=%d", msg)
	switch msg {
	case C.WM_CLOSE:
		C.DestroyWindow(hWnd)
	case C.WM_DESTROY:
		sn.tproxy.Raise(EventShutdownRequested)
	case C.WM_ENDSESSION:
		sn.tproxy.Raise(EventShutdownRequested)
	default:
		return C.DefWindowProc(hWnd, C.UINT(msg), wParam, lParam)
	}
	return 0
}
