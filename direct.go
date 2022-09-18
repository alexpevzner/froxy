//
// Direct transport
//

package main

import (
	"context"
	"net"
	"net/http"
)

//
// Transport that connects directly
//
type DirectTransport struct {
	http.Transport        // Direct http.Transport
	froxy          *Froxy // Back link to Froxy
}

//
// Create new DirectTransport
//
func NewDirectTransport(froxy *Froxy) *DirectTransport {
	t := &DirectTransport{
		Transport: http.Transport{
			Proxy:                 nil,
			MaxIdleConns:          HTTP_MAX_IDLE_CONNS,
			IdleConnTimeout:       HTTP_IDLE_CONN_TIMEOUT,
			ExpectContinueTimeout: HTTP_EXPECT_CONTINUE_TIMEOUT,
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

	return t.froxy.connMan.DialContext(ctx, network, addr,
		&t.froxy.Counters.TCPConnections)
}
