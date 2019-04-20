//
// Direct transport
//

package main

import (
	"context"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

//
// Transport that connects directly
//
type DirectTransport struct {
	http.Transport        // Direct http.Transport
	froxy          *Froxy // Back link to Froxy
}

//
// Direct TCP connection
//
type directConn struct {
	net.Conn                   // Underlying net.Conn
	closed    uint32           // Non-zero when closed
	transport *DirectTransport // Transport that owns the connection
}

//
// Create new DirectTransport
//
func NewDirectTransport(froxy *Froxy) *DirectTransport {
	t := &DirectTransport{
		Transport: http.Transport{
			Proxy:                 nil,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		froxy: froxy,
	}

	t.Transport.DialContext = t.DialContext

	return t
}

//
// Dial new TCP connection
//
func (t *DirectTransport) Dial(network, addr string) (net.Conn, error) {
	return t.DialContext(context.Background(), network, addr)
}

//
// Dial new TCP connection with context
//
func (t *DirectTransport) DialContext(ctx context.Context,
	network, addr string) (net.Conn, error) {

	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 10 * time.Second,
		DualStack: true,
	}

	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	t.froxy.IncCounter(&t.froxy.Counters.TCPConnections)

	dirconn := &directConn{
		Conn:      conn,
		transport: t,
	}

	return dirconn, nil
}

//
// Close directConn
//
func (conn *directConn) Close() error {
	var err error

	if atomic.SwapUint32(&conn.closed, 1) == 0 {
		t := conn.transport
		t.froxy.DecCounter(&t.froxy.Counters.TCPConnections)
		err = conn.Conn.Close()
	}

	return err

}