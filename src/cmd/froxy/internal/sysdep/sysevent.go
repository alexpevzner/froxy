//
// System events notifications
//

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
