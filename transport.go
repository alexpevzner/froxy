// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// Transport for outgoing connections

package main

import (
	"net"
	"net/http"
)

//
// Transport interface
//
type Transport interface {
	http.RoundTripper
	Dial(net, addr string) (net.Conn, error)
}
