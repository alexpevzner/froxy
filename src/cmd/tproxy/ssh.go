//
// SSH tunneling transport for net.http
//

package main

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

//
// The SSH transport for net.http
//
type SSHTransport struct {
	http.Transport                         // SSH-backed http.Transport
	server         string                  // Server address
	cfg            *ssh.ClientConfig       // SSH client configuration
	clients        map[*sshClient]struct{} // Pool of active clients
	mutex          sync.Mutex              // Access lock
}

var _ = Transport(&SSHTransport{})

//
// SSH client
//
type sshClient struct {
	*ssh.Client               // Underlying ssh.Client
	transport   *SSHTransport // Transport that owns the client
	refcnt      uint32        // Reference count
}

//
// SSH-tunneled connection
//
type sshConn struct {
	ssh.Conn            // Underlying ssh.Conn
	client   *sshClient // Client that owns the connection
}

//
// Create new SSH transport
//
func NewSSHTransport(server string, cfg *ssh.ClientConfig) *SSHTransport {
	t := &SSHTransport{
		Transport: http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		server:  server,
		cfg:     cfg,
		clients: make(map[*sshClient]struct{}),
	}

	t.Transport.Dial = t.Dial

	return t
}

//
// Dial new TCP connection, routed via server
//
func (t *SSHTransport) Dial(net, addr string) (net.Conn, error) {
	c, err := t.getClient()
	if err != nil {
		return nil, err
	}

	conn, err := c.Dial(net, addr)
	if err != nil {
		c.unref()
		return nil, err
	}

	return conn, nil
}

func (t *SSHTransport) getClient() (*sshClient, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	for c, _ := range t.clients {
		if c.refcnt < SSH_MAX_CONN_PER_CLIENT {
			c.refcnt++
			return c, nil
		}
	}

	sshclient, err := ssh.Dial("tcp", t.server, t.cfg)
	if err != nil {
		return nil, err
	}

	c := &sshClient{
		Client:    sshclient,
		transport: t,
		refcnt:    1,
	}

	t.clients[c] = struct{}{}
	go func() {
		sshclient.Wait()
		t.mutex.Lock()
		delete(t.clients, c)
		t.mutex.Unlock()
	}()

	return c, nil
}

// ----- sshClient methods -----
//
// Unref the client
//
func (c *sshClient) unref() {
	c.refcnt--
}

// ----- sshConn methods -----
//
// Close the connection
//
func (conn *sshConn) Close() error {
	err := conn.Conn.Close()
	conn.client.unref()
	return err
}
