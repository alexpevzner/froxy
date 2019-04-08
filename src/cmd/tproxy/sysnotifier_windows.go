//
// System events notifier -- Windows version
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
