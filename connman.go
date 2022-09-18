//
// Network connections manager
//

package main

import (
	"context"
	"net"
	"reflect"
	"sync/atomic"
	"unsafe"
)

//
// ConnMan manages all outgoing TCP connections
//
// It adds the following functionality to standard net.Conn connections:
//   1) Every connection is bound to its dial Context. When context
//      canceled, connection automatically closed
//   2) Changes in IP addresses, assigned to local interfaces, are
//      monitored. When connection's local address goes away, connection
//      automatically closed
//   3) It automatically manages statistics counter
//
type ConnMan struct {
	froxy        *Froxy           // Back link to Froxy
	dialer       net.Dialer       // The dialer
	cmd          chan interface{} // Command channel for ConnMan goroutine
	addrChgCount uint64           // Incremented on each EventIpAddrChanged
}

//
// Create new ConnMan
//
func NewConnMan(froxy *Froxy) *ConnMan {
	connman := &ConnMan{
		froxy: froxy,
		dialer: net.Dialer{
			KeepAlive: TCP_KEEP_ALIVE,
			DualStack: TCP_DUAL_STACK,
		},
		cmd: make(chan interface{}),
	}

	go connman.goroutine()

	return connman
}

//
// Dial new connection
//
func (connman *ConnMan) DialContext(ctx context.Context,
	network, addr string,
	counter *int32) (*Conn, error) {

	// Snapshot a addrChgCount
	addrChgCount := atomic.LoadUint64(&connman.addrChgCount)

	// Dial a connection
	c, err := connman.dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	// Create Conn
	conn := &Conn{
		Conn:    c,
		connman: connman,
		ctx:     ctx,
		counter: counter,
	}

	connman.froxy.IncCounter(counter)

	// Register a connection
	connman.cmd <- connManCmdAdd{conn: conn, addrChgCount: addrChgCount}

	return conn, nil
}

//
// Check local IP addresses and close connections
// that correspond to addresses not longer available
//
func (connman *ConnMan) recheckAddresses(byAddr map[string]map[*Conn]struct{}) {
	// Obtain all local addresses
	interfaces, err := net.Interfaces()
	if err != nil {
		connman.froxy.Error("Can't obtain local addresses: %s", err)
		return
	}

	all_addrs := make(map[string]struct{})
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			connman.froxy.Error("%s: cant't get addresses: %s", iface.Name, err)
			return
		}

		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if ok {
				ip := NetNormalizeIP(ipnet.IP)
				all_addrs[string(ip)] = struct{}{}
			}
		}
	}

	// Figure out connections without local addresses
	conns := make([]*Conn, 0, len(byAddr))
	for addr := range byAddr {
		_, ok := all_addrs[addr]
		if !ok {
			for conn := range byAddr[addr] {
				conns = append(conns, conn)
			}
		}
	}

	// Close all dead connections
	go func() {
		for _, conn := range conns {
			connman.froxy.Debug("local address gone, connection closed: %s/%s",
				conn.LocalAddr(), conn.RemoteAddr())
			conn.Abort(ErrNetDisconnected)
		}
	}()
}

//
// Connection manager goroutine
//
func (connman *ConnMan) goroutine() {
	events := connman.froxy.Sub(EventIpAddrChanged)
	defer connman.froxy.Unsub(events)

	byCtx := make(map[context.Context]map[*Conn]struct{})
	byAddr := make(map[string]map[*Conn]struct{})

	cases := make([]reflect.SelectCase, 0, 16)
	contexts := make([]context.Context, 0, 16)

	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(events),
	})

	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(connman.cmd),
	})

	contexts = append(contexts, nil, nil)

	for {
		// Refill array of reflect.Select() cases
		cases = cases[:2]
		contexts = contexts[:2]

		for ctx := range byCtx {
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(ctx.Done()),
			})

			contexts = append(contexts, ctx)
		}

		// Wait for some events
		num, data, _ := reflect.Select(cases)

		// Dispatch the event
		switch num {
		case 0:
			// Ebus event channel
			atomic.AddUint64(&connman.addrChgCount, 1)
			connman.recheckAddresses(byAddr)

		case 1:
			// Command channel
			switch cmd := data.Interface().(type) {
			case connManCmdAdd:
				// Add connection to byCtx map
				set := byCtx[cmd.conn.ctx]
				if set == nil {
					set = make(map[*Conn]struct{})
					byCtx[cmd.conn.ctx] = set
				}
				set[cmd.conn] = struct{}{}

				// Add connection to byAddr map
				addr, ok := cmd.conn.LocalAddr().(*net.TCPAddr)
				if ok {
					ip := NetNormalizeIP(addr.IP)
					set = byAddr[string(ip)]
					if set == nil {
						set = make(map[*Conn]struct{})
						byAddr[string(ip)] = set
					}
					set[cmd.conn] = struct{}{}
				}

				// Recheck addresses, if needed
				addrChgCount := atomic.LoadUint64(&connman.addrChgCount)
				if cmd.addrChgCount != addrChgCount {
					connman.recheckAddresses(byAddr)
				}

			case connManCmdDel:
				// Del connection from byCtx map
				set := byCtx[cmd.conn.ctx]
				if set != nil {
					delete(set, cmd.conn)
					if len(set) == 0 {
						delete(byCtx, cmd.conn.ctx)
					}
				}

				// Del connection from byAddr map
				addr, ok := cmd.conn.LocalAddr().(*net.TCPAddr)
				if ok {
					ip := NetNormalizeIP(addr.IP)
					set = byAddr[string(ip)]
					if set != nil {
						delete(set, cmd.conn)
						if len(set) == 0 {
							delete(byAddr, string(ip))
						}
					}
				}

			default:
				panic("internal error")
			}

		default:
			ctx := contexts[num]
			set := byCtx[ctx]
			if set != nil {
				delete(byCtx, ctx)
				go func() {
					for conn := range set {
						conn.Close()
					}
				}()
			}
		}
	}
}

// ----- Connection manager goroutine commands -----
//
// Add a connection
//
type connManCmdAdd struct {
	conn         *Conn  // The connection
	addrChgCount uint64 // addrChgCount snapshot just before dial
}

type connManCmdDel struct {
	conn *Conn // The connection
}

// ----- The managed connection -----
//
// Type Conn represents a single TCP connection
//
type Conn struct {
	net.Conn                 // Underlying connection
	connman  *ConnMan        // Back link to ConnMan
	ctx      context.Context // Connection's context
	counter  *int32          // Statistics counter
	errptr   unsafe.Pointer  // Close reason, *error
}

//
// Read from connection
//
func (conn *Conn) Read(b []byte) (n int, err error) {
	n, err = conn.Conn.Read(b)
	if err != nil {
		errptr := (*error)(atomic.LoadPointer(&conn.errptr))
		if errptr != nil && *errptr != nil {
			err = *errptr
		}
	}
	return
}

//
// Write to connection
//
func (conn *Conn) Write(b []byte) (n int, err error) {
	n, err = conn.Conn.Write(b)
	if err != nil {
		errptr := (*error)(atomic.LoadPointer(&conn.errptr))
		if errptr != nil && *errptr != nil {
			err = *errptr
		}
	}
	return
}

//
// Abort the connection
//
func (conn *Conn) Abort(reason error) error {
	var err error

	ok := atomic.CompareAndSwapPointer(&conn.errptr, nil,
		unsafe.Pointer(&reason))

	if ok {
		conn.connman.froxy.DecCounter(conn.counter)
		err = conn.Conn.Close()
		conn.connman.cmd <- connManCmdDel{conn: conn}
	}

	return err
}

//
// Close the connection
//
func (conn *Conn) Close() error {
	return conn.Abort(nil)
}
