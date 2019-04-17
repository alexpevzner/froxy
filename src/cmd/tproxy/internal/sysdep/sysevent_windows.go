//
// System events notifier -- Windows version
//

package sysdep

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
type SysEventNotifier struct {
	callback func(SysEvent)
	hWnd     C.HWND // Handle of hidden window for receiving system messages
}

//
// Create new SysEventNotifier
//
func NewSysEventNotifier(callback func(SysEvent)) *SysEventNotifier {
	sn := &SysEventNotifier{callback: callback}
	hWndChan := make(chan C.HWND)
	go sn.winGoroutine(hWndChan)
	sn.hWnd = <-hWndChan
	go sn.conGoroutine()
	return sn
}

//
// This goroutine waits for console events, using Go's os/signal wrapper
//
func (sn *SysEventNotifier) conGoroutine() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
	C.PostMessage(sn.hWnd, C.WM_CLOSE, 0, 0)
}

//
// This goroutine creates invisible window and waits for system messages
//
func (sn *SysEventNotifier) winGoroutine(hWndChan chan C.HWND) {
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
	hwnd := C.CreateWindowExA(
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

	hWndChan <- hwnd

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
// SysEventNotifier window procedure
//
func (sn *SysEventNotifier) wndProc(hWnd C.HWND, msg C.UINT, wParam C.WPARAM, lParam C.LPARAM) C.LRESULT {
	switch msg {
	case C.WM_CLOSE:
		C.DestroyWindow(hWnd)
	case C.WM_DESTROY, C.WM_QUIT, C.WM_ENDSESSION:
		sn.callback(SysEventShutdown)
	default:
		return C.DefWindowProc(hWnd, C.UINT(msg), wParam, lParam)
	}
	return 0
}
