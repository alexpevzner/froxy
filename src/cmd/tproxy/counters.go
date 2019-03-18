//
// Statistic counters
//

package main

//
// Collection of statistic counters
//
type Counters struct {
	Tag            uint64 `json:"tag"`
	TCPConnections int32  `json:"tcp_conns"`    // Direct TCP connections
	SSHSessions    int32  `json:"ssh_sessions"` // Count of SSH client sessions
	SSHConnections int32  `json:"ssh_conns"`    // Count of connections via SSH
}
