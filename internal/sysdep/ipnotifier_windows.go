// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// IP events notifier -- Windows version

package sysdep

/*
#undef _WIN32_WINNT
#define _WIN32_WINNT 0x0600
#include <winsock2.h>
#include <ws2ipdef.h>
#include <iphlpapi.h>

#cgo LDFLAGS: -l iphlpapi
*/
import "C"

import (
	"syscall"
	"unsafe"
)

//
// IP events notifier -- Windows version
//
type ipNotifier struct {
	hWnd             C.HWND   // Handle of hidden window for receiving system messages
	addrChangeHandle C.HANDLE // Address change subscription handle
}

var sysEventNotifierPtr *SysEventNotifier

//
// Initialize ipNotifier part of the SysEventNotifier
//
func (sn *SysEventNotifier) ipNotifierInit() {
	cb := syscall.NewCallback(ipNotifierCallback)

	// Note, Cgo doesn't allow us to pass sn pointer to
	// the NotifyUnicastIpAddressChange function as
	// callback parameter, so we simple save it into
	// the static pointer
	//
	// For now, we use only a single pointer per process,
	// so it is OK to use single pointer. But if we for some
	// reason will need to have multiple notifiers, a single
	// pointer will not work for us. So lets put check with
	// panic here, it will remind us to redesign this place
	// when we will actually need it
	if sysEventNotifierPtr != nil {
		panic("internal error")
	}

	status := C.NotifyUnicastIpAddressChange(
		C.AF_UNSPEC,
		C.PUNICAST_IPADDRESS_CHANGE_CALLBACK(unsafe.Pointer(cb)),
		C.PVOID(nil),
		C.FALSE,
		&sn.ipnotifier.addrChangeHandle,
	)

	if status != C.NO_ERROR {
		err := syscall.Errno(status)
		panic("NotifyUnicastIpAddressChange: " + err.Error())
	}
}

func ipNotifierCallback(ctx C.PVOID, r C.PMIB_UNICASTIPADDRESS_ROW, t C.MIB_NOTIFICATION_TYPE) int {
	sysEventNotifierPtr.callback(SysEventIpAddrChanged)
	return 0
}
