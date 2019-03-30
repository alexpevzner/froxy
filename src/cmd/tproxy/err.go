//
// Errors
//

package main

import (
	"errors"
)

// ----- TProxy errors -----
var (
	ErrTProxyRunning       = errors.New("TProxy already running")
	ErrCantKillTProxy      = errors.New("Can't kill running TProxy")
	ErrHttpHostMissed      = errors.New("invalid query: host parameter missed")
	ErrServerNotConfigured = errors.New("Server not configured")
)
