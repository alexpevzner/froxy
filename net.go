// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// Convenience utilities for networking

package main

import (
	"net"
	"strings"
)

//
// Split network address into host and port
//
// Unlike net.SplitHostPort, port allowed to be missed and
// will be substituted with defaultport at this case
//
func NetSplitHostPort(hostport, defaultport string) (host, port string) {
	if strings.IndexByte(hostport, ':') != -1 {
		var err error
		host, port, err = net.SplitHostPort(hostport)
		if err == nil {
			return
		}
	}

	return hostport, defaultport
}

//
// Provide default port, if port is missed in the address
//
func NetDefaultPort(hostport, defaultport string) string {
	host, port := NetSplitHostPort(hostport, defaultport)
	if port == "" {
		return host
	} else {
		return host + ":" + port
	}
}

//
// Normalize IP address. Returns 16-byte IP address for
// both IPv4 and IPv6 addresses
//
func NetNormalizeIP(ip net.IP) net.IP {
	if len(ip) == 4 {
		ip = net.IPv4(ip[0], ip[1], ip[2], ip[3])
	}
	return ip
}
