//
// Statistic counters
//

package main

//
// Collection of statistic counters
//
type Counters struct {
	Tag            uint64 `json:"tag"`
	SSHSessions    int32  `json:"ssh_sessions"` // Count of SSH client sessions
	SSHConnections int32  `json:"ssh_conns"`    // Count of connections via SSH
}
