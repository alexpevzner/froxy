// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// IP events notifier -- Linux version

package sysdep

// #include <linux/rtnetlink.h>
import "C"

import (
	"os"
	"syscall"
)

//
// IP events notifier -- Linux version
//
type ipNotifier struct {
	rtnetlink *os.File // rtnetlink socket wrapped into os.File
}

//
// Initialize ipNotifier part of the SysEventNotifier
//
func (sn *SysEventNotifier) ipNotifierInit() {
	// Open rtnetlink socket
	sock, err := syscall.Socket(syscall.AF_NETLINK,
		syscall.SOCK_RAW|syscall.SOCK_CLOEXEC,
		syscall.NETLINK_ROUTE)

	if err != nil {
		panic("AF_NETLINK open: " + err.Error())
	}

	// Subscribe to notifications
	var addr syscall.SockaddrNetlink
	addr.Family = syscall.AF_NETLINK
	addr.Groups = C.RTMGRP_IPV4_IFADDR | C.RTMGRP_IPV6_IFADDR

	err = syscall.Bind(sock, &addr)
	if err != nil {
		panic("AF_NETLINK bind: " + err.Error())
	}

	// Create ipNotifier structure
	sn.ipnotifier.rtnetlink = os.NewFile(uintptr(sock), "rtnetlink")

	// Start notification reader goroutine
	go sn.ipNotifierGoroutine()
}

//
// &ipNotifier goroutine
//
func (sn *SysEventNotifier) ipNotifierGoroutine() {
	buf := make([]byte, 16384)

	for {
		n, err := sn.ipnotifier.rtnetlink.Read(buf)
		if err != nil {
			panic("AF_NETLINK read: " + err.Error())
		}

		messages, err := syscall.ParseNetlinkMessage(buf[0:n])
		for _, msg := range messages {
			switch msg.Header.Type {
			case C.RTM_NEWADDR, C.RTM_DELADDR:
				sn.callback(SysEventIpAddrChanged)
			}

		}
	}
}
