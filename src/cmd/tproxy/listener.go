//
// Network listener
//

package main

import (
	"net"
	"sync/atomic"
	"time"
)

//
// The network listener
//
type Listener struct {
	tcplst  *net.TCPListener // Underlying net.TCPListener
	tcpaddr *net.TCPAddr     // Listener's address
	tproxy  *Tproxy          // Back link to Tproxy
}

//
// TCP local user connection, wrapped
//
type usertConn struct {
	net.Conn         // Underlying connection
	tproxy   *Tproxy // Back link to Tproxy
	closed   uint32  // Non-zero if closed
}

//
// Create new listener
//
func NewListener(tproxy *Tproxy, addr string) (*Listener, error) {
	// Resolve address
	tcpaddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	// Create TCPListener
	tcplst, err := net.ListenTCP("tcp", tcpaddr)
	if err != nil {
		return nil, err
	}

	// Create Listener structure
	return &Listener{tcplst, tcpaddr, tproxy}, nil
}

//
// Accept new connection
//
func (l *Listener) Accept() (net.Conn, error) {
	// Accept new TCP connection
	c, err := l.tcplst.AcceptTCP()
	if err != nil {
		return nil, err
	}

	// Setup TCP keep-alive
	c.SetKeepAlive(true)
	c.SetKeepAlivePeriod(3 * time.Minute)

	// Update statistics
	l.tproxy.IncCounter(&l.tproxy.Counters.UserConnections)

	// Wrap into usertConn structure
	return &usertConn{c, l.tproxy, 0}, nil
}

//
// Get listener's network address
//
func (l *Listener) Addr() net.Addr {
	return l.tcpaddr
}

//
// Close the listener
//
func (l *Listener) Close() error {
	return l.tcplst.Close()
}

//
// Close the connection
//
func (c *usertConn) Close() error {
	var err error
	if atomic.SwapUint32(&c.closed, 1) == 0 {
		c.tproxy.DecCounter(&c.tproxy.Counters.UserConnections)
		err = c.Conn.Close()
	}
	return err
}
