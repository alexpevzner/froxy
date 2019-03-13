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
	http.Transport
	env *Env // Back link to environment
}

//
// Create new DirectTransport
//
func NewDirectTransport(env *Env) *DirectTransport {
	return &DirectTransport{
		Transport: http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 10 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

//
// Dial new TCP connection
//
func (t *DirectTransport) Dial(net, addr string) (net.Conn, error) {
	return t.DialContext(context.Background(), net, addr)
}
