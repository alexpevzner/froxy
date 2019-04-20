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
	sessionsLock      sync.Mutex               // Access lock
	sessionsCond      *sync.Cond               // Wait queue for creating new sessions
	sessions          map[*sshSession]struct{} // Pool of active sessions
	sessionsConnCount int                      // Count of connections, active+planned
	sessionsCount     int                      // Count of sessions, active+planned

	// Disconnect/reconnect machinery
	reconnectLock       sync.Mutex         // Reconnect lock
	disconnectLock      sync.Mutex         // Disconnect machinery lock
	disconnectWait      sync.WaitGroup     // To wait for disconnect completion
	disconnectCtx       context.Context    // To break dial-in-progress
	disconnectCtxCancel context.CancelFunc // To cancel disconnectCtx
}

var _ = Transport(&SSHTransport{})

//
// SSH session -- wraps ssh.Client
//
type sshSession struct {
	*ssh.Client               // Underlying ssh.Client
	transport   *SSHTransport // Transport that owns the session
	refcnt      uint32        // Reference count
}

//
// SSH-tunneled connection
//
type sshConn struct {
	net.Conn             // Underlying SSH-backed net.Conn
	closed   uint32      // Non-zero when closed
	session  *sshSession // Session that owns the connection
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
		froxy:    froxy,
		sessions: make(map[*sshSession]struct{}),
	}

	t.sessionsCond = sync.NewCond(&t.sessionsLock)
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
	// Only one Reconnect() in time allowed
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

	// Obtain SSH session
	session, err := t.getSession(ctx, params, key)
	if err != nil {
		t.disconnectWait.Done()
		err = fmt.Errorf("Can't connect to the server %q: %s", params.Addr, err)
		return nil, err
	}

	// Dial a new connection
	conn, err := session.Dial(net, addr)
	if err != nil {
		session.unref()
		err = fmt.Errorf("Server can't connect to %q: %s", addr, err)
		return nil, err
	}

	t.froxy.Debug("SSH: connection established")
	t.froxy.IncCounter(&t.froxy.Counters.SSHConnections)

	return &sshConn{Conn: conn, session: session}, nil
}

//
// Get a session for establishing new connection
//
// Either reuses a spare session, if available, or dials
// a new session on demand
//
func (t *SSHTransport) getSession(
	ctx context.Context,
	params ServerParams,
	key *keys.Key) (*sshSession, error) {

	// Acquire the lock
	t.sessionsLock.Lock()
	defer t.sessionsLock.Unlock()
	defer t.sessionsCond.Signal()

	// Update counters
	t.sessionsConnCount++

AGAIN:
	// Lookup a spare session
	session := t.spareSession()
	if session != nil {
		return session, nil
	}

	// Wait until opportunity to create new session
	if t.sessionsCount*SSH_MAX_CONN_PER_CLIENT >= t.sessionsConnCount {
		t.sessionsCond.Wait()
		goto AGAIN
	}

	// Update counters
	t.sessionsCount++

	// Dial a new session
	t.sessionsLock.Unlock()
	session, err := t.newSession(ctx, params, key)
	t.sessionsLock.Lock()

	if err == nil {
		return session, nil
	}

	// Cleanup after error
	t.sessionsConnCount--
	t.sessionsCount--

	return nil, err
}

//
// Find a spare session for establishing new connection
// May return nil if appropriate session is not found
//
// MUST be called under t.sessionsLock
//
func (t *SSHTransport) spareSession() *sshSession {
	session := (*sshSession)(nil)

	for ssn, _ := range t.sessions {
		if ssn.refcnt < SSH_MAX_CONN_PER_CLIENT {
			if session == nil || session.refcnt > ssn.refcnt {
				session = ssn
			}
		}
	}

	if session != nil {
		session.refcnt++
	}

	return session
}

//
// Establish a new client session
//
func (t *SSHTransport) newSession(
	ctx context.Context,
	params ServerParams,
	key *keys.Key) (*sshSession, error) {

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

	// Dial a new network connection
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

	// Create &sshSession structure
	session := &sshSession{
		Client:    ssh.NewClient(c, chans, reqs),
		transport: t,
		refcnt:    1,
	}

	t.sessionsLock.Lock()
	t.froxy.IncCounter(&t.froxy.Counters.SSHSessions)
	t.sessions[session] = struct{}{}
	t.sessionsLock.Unlock()

	// Wait in background for connection termination
	go func() {
		err := session.Wait()

		t.sessionsLock.Lock()

		delete(t.sessions, session)

		t.froxy.DecCounter(&t.froxy.Counters.SSHSessions)
		t.sessionsCount--
		if t.sessionsCount == 0 && ctx.Err() == nil {
			t.froxy.SetConnState(ConnTrying, err.Error())
		}

		t.sessionsLock.Unlock()

		t.sessionsCond.Signal()
		t.disconnectWait.Done()
	}()

	return session, nil
}

// ----- sshSession methods -----
//
// Unref the session
//
func (ssn *sshSession) unref() {
	t := ssn.transport

	t.sessionsLock.Lock()

	ssn.refcnt--

	t.sessionsConnCount--
	t.sessionsCond.Signal()

	t.sessionsLock.Unlock()
}

// ----- sshConn methods -----
//
// Close the connection
//
func (conn *sshConn) Close() error {
	var err error

	if atomic.SwapUint32(&conn.closed, 1) == 0 {
		t := conn.session.transport

		t.froxy.Debug("SSH: connection closed")
		err = conn.Conn.Close()

		t.froxy.DecCounter(&t.froxy.Counters.SSHConnections)
		conn.session.unref()
	}

	return err
}
