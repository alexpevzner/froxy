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

void ipNotifierCallback(PVOID, PMIB_UNICASTIPADDRESS_ROW, MIB_NOTIFICATION_TYPE);

#cgo LDFLAGS: -l iphlpapi
*/
import "C"

import (
	"syscall"
	"unsafe"
)

//
// Initialize ipNotifier part of the SysEventNotifier
//
func (sn *SysEventNotifier) ipNotifierInit() {
	status := C.NotifyUnicastIpAddressChange(
		C.AF_UNSPEC,
		C.PUNICAST_IPADDRESS_CHANGE_CALLBACK(C.ipNotifierCallback),
		C.PVOID(unsafe.Pointer(sn)),
		C.FALSE,
		&sn.ipnotifier.addrChangeHandle,
	)

	if status != C.NO_ERROR {
		err := syscall.Errno(status)
		panic("NotifyUnicastIpAddressChange: " + err.Error())
	}
}

//export ipNotifierCallback
func ipNotifierCallback(ctx C.PVOID, r C.PMIB_UNICASTIPADDRESS_ROW, t C.MIB_NOTIFICATION_TYPE) {
	sn := (*SysEventNotifier)(unsafe.Pointer(ctx))
	sn.callback(SysEventIpAddrChanged)
}
