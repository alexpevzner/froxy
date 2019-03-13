//
// Convenience utilities for networking
//

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
