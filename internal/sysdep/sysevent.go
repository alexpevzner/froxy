// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// System events notifications, common part

package sysdep

//
// System events
//
type SysEvent int

const (
	SysEventShutdown = iota
	SysEventIpAddrChanged
)

//
// System events notifier
//
type SysEventNotifier struct {
	callback   func(SysEvent) // Event callback
	ipnotifier ipNotifier     // System-specific IP events notifier
}
