//
// SSH tunneling transport for net.http
//

package main

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

//
// The SSH transport for net.http
//
type SSHTransport struct {
	http.Transport                         // SSH-backed http.Transport
	env            *Env                    // Back link to environment
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
	net.Conn            // Underlying SSH-backed net.Conn
	client   *sshClient // Client that owns the connection
}

//
// Create new SSH transport
//
func NewSSHTransport(env *Env, cfg *ssh.ClientConfig) *SSHTransport {
	t := &SSHTransport{
		Transport: http.Transport{
			Proxy:                 nil,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		env:     env,
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

	t.env.Debug("SSS: connection established")
	atomic.AddInt32(&t.env.Counters.SSHConnections, 1)

	return &sshConn{Conn: conn, client: c}, nil
}

//
// Get a client session for establishing new connection
//
func (t *SSHTransport) getClient() (*sshClient, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Look to existent clients
	clnt := (*sshClient)(nil)
	for c, _ := range t.clients {
		if c.refcnt < SSH_MAX_CONN_PER_CLIENT {
			if clnt == nil || clnt.refcnt > c.refcnt {
				clnt = c
			}
		}
	}

	if clnt != nil {
		clnt.refcnt++
		return clnt, nil
	}

	// Create a new one
	params := t.env.GetServerParams()
	t.env.Debug("params=%#v)", params)

	if !params.Configured() {
		t.env.SetConnState(ConnNotConfigured, "")
		return nil, errors.New("Server not configured")
	}

	cfg := &ssh.ClientConfig{
		User: params.Login,
		Auth: []ssh.AuthMethod{
			ssh.Password(params.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshclient, err := ssh.Dial("tcp", NetDefaultPort(params.Addr, "22"), cfg)
	if err != nil {
		if len(t.clients) == 0 {
			t.env.SetConnState(ConnTrying, err.Error())
		}

		return nil, err
	}

	t.env.SetConnState(ConnEstablished, "")

	clnt = &sshClient{
		Client:    sshclient,
		transport: t,
		refcnt:    1,
	}

	atomic.AddInt32(&t.env.Counters.SSHSessions, 1)
	t.clients[clnt] = struct{}{}

	go func() {
		err := sshclient.Wait()

		t.mutex.Lock()

		atomic.AddInt32(&t.env.Counters.SSHSessions, -1)
		delete(t.clients, clnt)

		if len(t.clients) == 0 {
			t.env.SetConnState(ConnTrying, err.Error())
		}

		t.mutex.Unlock()
	}()

	return clnt, nil
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
	conn.client.transport.env.Debug("SSS: connection closed")

	atomic.AddInt32(&conn.client.transport.env.Counters.SSHConnections, -1)
	err := conn.Conn.Close()
	conn.client.unref()
	return err
}
