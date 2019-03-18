//
// Direct transport
//

package main

import (
	"context"
	"net"
	"net/http"
	"time"
)

//
// Transport that connects directly
//
type DirectTransport struct {
	http.Transport      // Direct http.Transport
	env            *Env // Back link to environment
}

//
// Direct TCP connection
//
type directConn struct {
	net.Conn                   // Underlying net.Conn
	transport *DirectTransport // Transport that owns the connection
}

//
// Create new DirectTransport
//
func NewDirectTransport(env *Env) *DirectTransport {
	t := &DirectTransport{
		Transport: http.Transport{
			Proxy:                 nil,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		env: env,
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

	t.env.IncCounter(&t.env.Counters.TCPConnections)

	dirconn := &directConn{
		Conn:      conn,
		transport: t,
	}

	return dirconn, nil
}

//
// Close directConn
//
func (c *directConn) Close() error {
	t := c.transport
	t.env.DecCounter(&t.env.Counters.TCPConnections)
	return c.Conn.Close()
}
