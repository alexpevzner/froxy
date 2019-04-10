//
// Statistic counters
//

package main

//
// Collection of statistic counters
//
type Counters struct {
	UserConnections int32 `json:"user_conns"`        // Local user connections
	TCPConnections  int32 `json:"tcp_conns"`         // Direct TCP connections
	SSHSessions     int32 `json:"ssh_sessions"`      // Count of SSH client sessions
	SSHConnections  int32 `json:"ssh_conns"`         // Count of connections via SSH
	HTTPRqReceived  int32 `json:"http_rq_received"`  // Total count of received requests
	HTTPRqPending   int32 `json:"http_rq_pending"`   // Count of pending requests
	HTTPRqDirect    int32 `json:"http_rq_direct"`    // Count of direct requests
	HTTPRqForwarded int32 `json:"http_rq_forwarded"` // Count of forwarded requests
	HTTPRqBlocked   int32 `json:"http_rq_blocked"`   // Count of blocked requests
}
