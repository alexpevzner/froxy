// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// Network listener

package main

import (
	"net"
	"sync/atomic"
)

//
// The network listener
//
type Listener struct {
	tcplst  *net.TCPListener // Underlying net.TCPListener
	tcpaddr *net.TCPAddr     // Listener's address
	froxy   *Froxy           // Back link to Froxy
}

//
// TCP local user connection, wrapped
//
type usertConn struct {
	net.Conn        // Underlying connection
	froxy    *Froxy // Back link to Froxy
	closed   uint32 // Non-zero if closed
}

//
// Create new listener
//
func NewListener(froxy *Froxy, addr string) (*Listener, error) {
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
	return &Listener{tcplst, tcpaddr, froxy}, nil
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
	//
	// FIXME. Seems, we don't need keep-alive at server
	// side, because the server only works on a localhost
	// where TCP keep-alive is meaningless
	//
	//c.SetKeepAlive(true)
	//c.SetKeepAlivePeriod(TCP_KEEP_ALIVE)

	// Update statistics
	l.froxy.IncCounter(&l.froxy.Counters.UserConnections)

	// Wrap into usertConn structure
	return &usertConn{c, l.froxy, 0}, nil
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
		c.froxy.DecCounter(&c.froxy.Counters.UserConnections)
		err = c.Conn.Close()
	}
	return err
}
