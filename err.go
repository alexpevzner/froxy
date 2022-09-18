// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// Errors

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
	ErrNoSuchKey           = errors.New("Now such key")
	ErrSiteBlocked         = errors.New("Site blocked")
	ErrNetDisconnected     = errors.New("Disconnected from the network")
)
