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
	http.Transport             // SSH-backed http.Transport
	froxy          *Froxy      // Back link to Froxy
	ctx            *sshContext // Current context

	// Management of active sessions
	sessionsLock      sync.Mutex               // Access lock
	sessionsCond      *sync.Cond               // Wait queue for creating new sessions
	sessions          map[*sshSession]struct{} // Pool of active sessions
	sessionsConnCount int                      // Count of connections, active+planned
	sessionsCount     int                      // Count of sessions, active+planned

	// Disconnect/reconnect machinery
	disconnectLock sync.RWMutex   // Disconnect machinery lock
	disconnectWait sync.WaitGroup // To wait for disconnect completion
}

var _ = Transport(&SSHTransport{})

// ----- SSH connection context -- wraps context.Context -----
//
// SSH connection context
//
type sshContext struct {
	context.Context                    // Underlying context
	cancel          context.CancelFunc // Context cancel function
	froxy           *Froxy             // Back link to Froxy
	params          *ServerParams      // Server parameters
	key             *keys.Key          // SSH key to use, if any
	ok              bool               // Server parameters OK to connect
}

//
// Create new sshContext
//
func newSshContext(froxy *Froxy, params *ServerParams) *sshContext {
	ctx := &sshContext{
		froxy:  froxy,
		params: params,
		ok:     params.Addr != "" && params.Login != "",
	}

	if ctx.ok {
		if params.Keyid != "" {
			ctx.key = ctx.froxy.KeyById(params.Keyid)
			ctx.ok = ctx.key != nil
		} else {
			ctx.ok = params.Password != ""
		}
	}

	ctx.Context, ctx.cancel = context.WithCancel(context.Background())

	return ctx
}

//
// Cancel the context
//
func (ctx *sshContext) Cancel() {
	ctx.cancel()
}

//
// Check of server parameters are equal to those associated
// with the context
//
func (ctx *sshContext) ServerParamsEqual(params *ServerParams) bool {
	return reflect.DeepEqual(ctx.params, params)
}

//
// Create SSH client configuration
//
func (ctx *sshContext) SshClientConfig() *ssh.ClientConfig {
	var auth []ssh.AuthMethod
	if ctx.key != nil {
		auth = []ssh.AuthMethod{ssh.PublicKeys(ctx.key.Signer())}
	} else {
		auth = []ssh.AuthMethod{ssh.Password(ctx.params.Password)}
	}

	return &ssh.ClientConfig{
		User:            ctx.params.Login,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

// ----- SSH session -----
//
// SSH session -- wraps ssh.Client
//
type sshSession struct {
	*ssh.Client               // Underlying ssh.Client
	transport   *SSHTransport // Transport that owns the session
	refcnt      uint32        // Reference count
}

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

// ----- SSH-tunneled connection -----
//
// SSH-tunneled connection
//
type sshConn struct {
	net.Conn             // Underlying SSH-backed net.Conn
	closed   uint32      // Non-zero when closed
	session  *sshSession // Session that owns the connection
}

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

// ----- SSHTransport methods -----
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
	// Synchronize with disconnect logic
	t.disconnectLock.Lock()
	defer t.disconnectLock.Unlock()

	// Something changed?
	if t.ctx != nil && t.ctx.ServerParamsEqual(&params) {
		return
	}

	// Disconnect if we were connected
	if t.ctx != nil && t.ctx.ok {
		t.ctx.Cancel()
		t.ctx = nil
		t.disconnectWait.Wait()
	}

	t.ctx = newSshContext(t.froxy, &params)

	// Update connection state
	if t.ctx.ok {
		t.froxy.SetConnState(ConnTrying, "")
	} else {
		t.froxy.SetConnState(ConnNotConfigured, "")
	}
}

//
// Dial new TCP connection, routed via server
//
func (t *SSHTransport) Dial(net, addr string) (net.Conn, error) {
	// Synchronize with disconnect logic
	t.disconnectWait.Add(1)
	defer t.disconnectWait.Done()

	t.disconnectLock.RLock()
	ctx := t.ctx
	t.disconnectLock.RUnlock()

	if !ctx.ok {
		return nil, ErrServerNotConfigured
	}

	// Obtain SSH session
	session, err := t.getSession(ctx)
	if err != nil {
		err = fmt.Errorf("Can't connect to the server %q: %s", ctx.params.Addr, err)
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
func (t *SSHTransport) getSession(ctx *sshContext) (*sshSession, error) {
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
	session, err := t.newSession(ctx)
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
func (t *SSHTransport) newSession(ctx *sshContext) (*sshSession, error) {
	// Create SSH configuration
	cfg := ctx.SshClientConfig()

	// Dial a new network connection
	dialer := &net.Dialer{Timeout: cfg.Timeout}
	addr := NetDefaultPort(ctx.params.Addr, "22")
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

	t.disconnectWait.Add(1)

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
