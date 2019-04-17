//
// SSH tunneling transport for net.http
//

package main

import (
	"cmd/froxy/internal/keys"
	"context"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

//
// The SSH transport for net.http
//
type SSHTransport struct {
	http.Transport                 // SSH-backed http.Transport
	froxy             *Froxy       // Back link to Froxy
	params            ServerParams // Server parameters
	key               *keys.Key    // SSH key to use, if any
	paramsOkToConnect bool         // Server parameters OK to connect

	// Management of active sessions
	clientsLock sync.Mutex              // Access lock
	clients     map[*sshClient]struct{} // Pool of active clients

	// Establishing new sessions
	newClientLock      sync.Mutex // Access lock
	newClientCond      *sync.Cond // Wait queue
	newClientWaitCount int        // Count of waiters
	newClientDialCount int        // Count of active dialers

	// Disconnect/reconnect machinery
	reconnectLock       sync.Mutex         // Reconnect lock
	disconnectLock      sync.Mutex         // Disconnect machinery lock
	disconnectWait      sync.WaitGroup     // To wait for disconnect completion
	disconnectCtx       context.Context    // To break dial-in-progress
	disconnectCtxCancel context.CancelFunc // To cancel disconnectCtx
}

var _ = Transport(&SSHTransport{})

//
// SSH client -- wraps ssh.Client
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
	closed   uint32     // Non-zero when closed
	client   *sshClient // Client that owns the connection
}

//
// Create new SSH transport
//
func NewSSHTransport(froxy *Froxy) *SSHTransport {
	t := &SSHTransport{
		Transport: http.Transport{
			Proxy:                 nil,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		froxy:   froxy,
		clients: make(map[*sshClient]struct{}),
	}

	t.newClientCond = sync.NewCond(&t.newClientLock)
	t.Transport.Dial = func(net, addr string) (net.Conn, error) {
		conn, err := t.Dial(net, addr)
		return conn, err
	}

	t.Reconnect(t.froxy.GetServerParams())

	return t
}

//
// Reconnect to the server
//
// This function updates server connection parameters, which
// may either cause a disconnect or [re]connect
//
// In a case of [re]connect this function doesn't establish server
// connection immediately, it only initiates asynchronous process of
// establishing server connection
//
// In a case of disconnect, this function synchronously waits until
// all active connections has gone away
//
func (t *SSHTransport) Reconnect(params ServerParams) {
	// Only one Reconnect () in time allowed
	t.reconnectLock.Lock()
	defer t.reconnectLock.Unlock()

	// Synchronize with disconnect logic
	t.disconnectLock.Lock()

	// Something changed?
	if reflect.DeepEqual(t.params, params) {
		t.disconnectLock.Unlock()
		return
	}

	// Cancel previous connection and reconnect
	if t.disconnectCtxCancel != nil {
		t.disconnectCtxCancel()
	}

	t.disconnectCtx, t.disconnectCtxCancel = context.WithCancel(
		context.Background())
	t.params = params

	// Check that we have everything to proceed with connection
	t.paramsOkToConnect = params.Addr != "" && params.Login != ""
	if t.paramsOkToConnect {
		if params.Keyid != "" {
			t.key = t.froxy.KeyById(params.Keyid)
			t.paramsOkToConnect = t.key != nil
		} else {
			t.paramsOkToConnect = params.Password != ""
		}
	}

	// Update connection state
	if t.paramsOkToConnect {
		t.froxy.SetConnState(ConnTrying, "")
	} else {
		t.froxy.SetConnState(ConnNotConfigured, "")
	}

	t.disconnectLock.Unlock()

	// In a case of disconnect -- wait for completion
	if !t.paramsOkToConnect {
		t.disconnectWait.Wait()
	}
}

//
// Dial new TCP connection, routed via server
//
func (t *SSHTransport) Dial(net, addr string) (net.Conn, error) {
	// Synchronize with disconnect logic
	t.disconnectLock.Lock()

	ctx := t.disconnectCtx
	params := t.params
	key := t.key
	var err error

	if t.paramsOkToConnect {
		t.disconnectWait.Add(1)
	} else {
		err = ErrServerNotConfigured
	}

	t.disconnectLock.Unlock()

	// Can connect?
	if err != nil {
		return nil, err
	}

	// Obtain SSH client
	clnt := t.getClient()
	if clnt == nil {
		clnt, err = t.newClient(ctx, params, key)
	}
	if err != nil {
		t.disconnectWait.Done()
		err = fmt.Errorf("Can't connect to the server %q: %s",
			params.Addr, err)
		return nil, err
	}

	// Dial a new connection
	conn, err := clnt.Dial(net, addr)
	if err != nil {
		clnt.unref()
		err = fmt.Errorf("Server can connect to %q: %s",
			addr, err)
		return nil, err
	}

	t.froxy.Debug("SSH: connection established")
	t.froxy.IncCounter(&t.froxy.Counters.SSHConnections)

	return &sshConn{Conn: conn, client: clnt}, nil
}

//
// Get a client session for establishing new connection
// May return nil if appropriate session is not found
//
func (t *SSHTransport) getClient() *sshClient {
	// Acquire the lock
	t.clientsLock.Lock()
	t.clientsLock.Unlock()

	// Lookup a client pool
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
	}

	return clnt
}

// ----- Establishing new client sessions -----
//
// Wait until opportunity to establish a new session
//
// This function effectively limits a rate of establishing
// new sessions, depending on a present demand in new connections
// (one session my provide up to SSH_MAX_CONN_PER_CLIENT connections)
//
// Usage:
//     t.newClientWait()
//     c := t.getClient()
//     if c == nil {
//         c = dialNewClient(...)
//     }
//     t.newClientDone()
//
func (t *SSHTransport) newClientWait() {
	t.newClientLock.Lock()
	t.newClientWaitCount++

	for t.newClientDialCount*SSH_MAX_CONN_PER_CLIENT >= t.newClientWaitCount {
		t.newClientCond.Wait()
	}

	t.newClientDialCount++
	t.newClientLock.Unlock()
}

//
// Notify new sessions scheduler that session establishment is
// done (or not longer needed)
//
func (t *SSHTransport) newClientDone() {
	t.newClientLock.Lock()
	t.newClientDialCount--
	t.newClientWaitCount--
	t.newClientCond.Signal()
	t.newClientLock.Unlock()
}

//
// Establish a new client connection
//
func (t *SSHTransport) newClient(
	ctx context.Context,
	params ServerParams,
	key *keys.Key) (*sshClient, error) {

	// Obtain grant to dial new session
	t.newClientWait()
	defer t.newClientDone()

	// Do we still need to dial?
	clnt := t.getClient()
	if clnt != nil {
		return clnt, nil
	}

	// Create SSH configuration
	t.froxy.Debug("params=%#v)", params)

	auth := []ssh.AuthMethod{}
	if key != nil {
		auth = append(auth, ssh.PublicKeys(key.Signer()))
	} else {
		auth = append(auth, ssh.Password(params.Password))
	}

	cfg := &ssh.ClientConfig{
		User:            params.Login,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Dial new network connection
	dialer := &net.Dialer{Timeout: cfg.Timeout}
	addr := NetDefaultPort(params.Addr, "22")
	conn, err := dialer.DialContext(ctx, "tcp", addr)

	if err != nil {
		return nil, err
	}

	// Make sure connection closes when ctx is cancelled
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	// Perform SSH handshake
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		return nil, err
	}

	t.froxy.SetConnState(ConnEstablished, "")

	// Create &sshClient structure
	clnt = &sshClient{
		Client:    ssh.NewClient(c, chans, reqs),
		transport: t,
		refcnt:    1,
	}

	t.clientsLock.Lock()
	t.froxy.IncCounter(&t.froxy.Counters.SSHSessions)
	t.clients[clnt] = struct{}{}
	t.clientsLock.Unlock()

	// Wait in background for connection termination
	go func() {
		err := clnt.Wait()

		t.clientsLock.Lock()

		t.froxy.DecCounter(&t.froxy.Counters.SSHSessions)
		delete(t.clients, clnt)

		if len(t.clients) == 0 && ctx.Err() == nil {
			t.froxy.SetConnState(ConnTrying, err.Error())
		}

		t.disconnectWait.Done()

		t.clientsLock.Unlock()
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
	var err error

	if atomic.SwapUint32(&conn.closed, 1) == 0 {
		t := conn.client.transport

		t.froxy.Debug("SSH: connection closed")

		t.froxy.DecCounter(&t.froxy.Counters.SSHConnections)
		err = conn.Conn.Close()
		conn.client.unref()
	}

	return err
}
