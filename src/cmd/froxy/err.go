//
// Errors
//

package main

import (
	"errors"
)

// ----- Froxy errors -----
var (
	ErrFroxyRunning        = errors.New("Froxy already running")
	ErrCantKillFroxy       = errors.New("Can't kill running Froxy")
	ErrHttpHostMissed      = errors.New("invalid query: host parameter missed")
	ErrServerNotConfigured = errors.New("Server not configured")
	ErrKeyIdMissed         = errors.New("invalid query: key ID missed")
	ErrNoSuchKey           = errors.New("Now suck key")
	ErrSiteBlocked         = errors.New("Site blocked")
)
