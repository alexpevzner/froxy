//
// Transport for outgoing connections
//

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
