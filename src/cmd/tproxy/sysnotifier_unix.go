//
// System events notifier -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package main

import (
	"os"
	"os/signal"
	"syscall"
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
	go sn.goroutine()
	return sn
}

//
// SysNotifier goroutine
//
func (sn *SysNotifier) goroutine() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGHUP)

	s := <-c
	sn.tproxy.Debug("Signal %s received", s)
	sn.tproxy.Raise(EventShutdownRequested)
}
