//
// System events notifier -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package main

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
	return &SysNotifier{}
}
