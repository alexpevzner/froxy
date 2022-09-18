//
// System events notifier -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris
//

package sysdep

import (
	"os"
	"os/signal"
	"syscall"
)

//
// Create new SysEventNotifier
//
func NewSysEventNotifier(callback func(SysEvent)) *SysEventNotifier {
	sn := &SysEventNotifier{callback: callback}
	sn.ipNotifierInit()
	go sn.goroutine()
	return sn
}

//
// SysEventNotifier goroutine
//
func (sn *SysEventNotifier) goroutine() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGHUP)

	<-c
	sn.callback(SysEventShutdown)
}
