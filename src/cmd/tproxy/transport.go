//
// Transport for outgoing connections
//

package main

import (
	"net"
	"net/http"

	"golang.org/x/crypto/ssh"
)

//
// Transport interface
//
type Transport interface {
	http.RoundTripper
	Dial(net, addr string) (net.Conn, error)
}

var sshTransport = NewSSHTransport(
	"cikorio.com:22",
	&ssh.ClientConfig{
		User: "proxy",
		Auth: []ssh.AuthMethod{
			ssh.Password("proxy12345"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	},
)

var directTransport = NewDirectTransport()
