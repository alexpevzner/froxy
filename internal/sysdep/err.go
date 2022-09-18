// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// Errors

package sysdep

import (
	"errors"
)

var (
	ErrLockIsBusy = errors.New("Lock is busy")
)
