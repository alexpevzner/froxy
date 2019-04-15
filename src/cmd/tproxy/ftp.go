//
// ftp proxy
//

package main

import (
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jlaffaye/ftp"
)

type FTPProxy struct {
	tproxy          *Tproxy                       // Back link to tproxy
	idleLock        sync.Mutex                    // Protects idle connections machinery
	idleConnections map[string]ftpIddleConnBucket // Per-site buckets of idle connections
	idleChan        chan struct{}                 // Signaling channel for idle expiration goroutine
}

//
// Type ftpIddleConnBucket represents a bucket of
// idle connections to the same site
//
type ftpIddleConnBucket map[*ftpConn]time.Time

//
// FTP connection
//
type ftpConn struct {
	*ftp.ServerConn           // Underlying ftp.ServerConn
	ftpp            *FTPProxy // Back link to owning FTPProxy
	netconn         net.Conn  // Underlying net.Conn
	site            string    // Site URL
	reused          bool      // This is reused idle connection
	closed          bool      // This is closed connection
}

//
// Create new FTP proxy
//
func NewFTPProxy(tproxy *Tproxy) *FTPProxy {
	ftpp := &FTPProxy{
		tproxy:          tproxy,
		idleConnections: make(map[string]ftpIddleConnBucket),
		idleChan:        make(chan struct{}),
	}

	go ftpp.expireIdleConnections()
	return ftpp
}

//
// Handle a single "HTTP" GET request
//
func (ftpp *FTPProxy) Handle(w http.ResponseWriter, r *http.Request, transport Transport) {
	// Check protocol and method
	if r.URL.Scheme != "ftp" {
		ftpp.tproxy.httpError(w, http.StatusServiceUnavailable,
			fmt.Errorf("unsupported protocol scheme %q", r.URL.Scheme))
		return
	}

	if r.Method != "GET" {
		ftpp.tproxy.httpError(w, http.StatusMethodNotAllowed,
			fmt.Errorf("unsupported method %q", r.Method))
		return
	}

	// Create a copy of URL that contains only parts related to site
	site_url := url.URL{
		Scheme: r.URL.Scheme,
		User:   r.URL.User,
		Host:   r.URL.Host,
	}

	site := site_url.String()

	// Normalize path
	path := r.URL.Path
	isDir := strings.HasSuffix(path, "/")
	if isDir && len(path) > 1 {
		path = path[:len(path)-1]
	}

	// Obtain a connection
RETRY:
	conn := ftpp.getConn(site)
	var err error
	if conn == nil {
		conn, err = ftpp.dialConn(transport, site_url)
		if err != nil {
			ftpp.sendError(w, 0, err)
			return
		}
	}

	defer conn.putConn()

	// Try to interpret path as file
	httpStatus := 0
	if !isDir {
		var body *ftp.Response
		body, httpStatus, err = ftpp.readFile(conn, path)
		if err == nil {
			ftpp.sendFile(w, body)
			return
		}
	}

	// Try to interpret path as directory
	if list, httpStatus2, err2 := ftpp.readDir(conn, path); err2 == nil {
		ftpp.sendDirectory(w, r, path, list)
		return
	} else {
		if err == nil {
			err = err2
			httpStatus = httpStatus2
		}
	}

	// If it is reused connection, try to reconnect
	conn.Close()
	if conn.reused {
		goto RETRY
	}

	// Error handling
	if httpStatus == 0 {
		httpStatus = http.StatusServiceUnavailable
	}

	ftpp.sendError(w, httpStatus, err)
}

//
// Read the file
//
func (ftpp *FTPProxy) readFile(conn *ftpConn, path string) (*ftp.Response, int, error) {
	// Try to fetch the file
	ftpp.tproxy.Debug("FTP: RETR %q", path)
	body, err := conn.Retr(path)
	if err == nil {
		return body, 0, nil
	} else {
		ftpp.tproxy.Debug("FTP: RETR: %s", err)
	}

	// Try to guess appropriate HTTP status
	httpStatus := 0
	if ftperr, ok := err.(*textproto.Error); ok && ftperr.Code == ftp.StatusFileUnavailable {
		ftpp.tproxy.Debug("FTP: SIZE %q", path)
		_, err2 := conn.FileSize(path)
		if err2 == nil {
			httpStatus = http.StatusForbidden
		} else {
			ftpp.tproxy.Debug("FTP: SIZE: %s", err2)
			httpStatus = http.StatusNotFound
		}
	}

	return nil, httpStatus, err
}

//
// Read the directory
//
func (ftpp *FTPProxy) readDir(conn *ftpConn, path string) ([]*ftp.Entry, int, error) {
	ftpp.tproxy.Debug("FTP: CWD %q", path)
	err := conn.ChangeDir(path)
	if err != nil {
		ftpp.tproxy.Debug("FTP: CWD %s", err)
		return nil, 0, err
	}

	ftpp.tproxy.Debug("FTP: LIST .")
	files, err := conn.List(".")
	if err != nil {
		ftpp.tproxy.Debug("FTP: LIST %s", err)
	}

	ftpp.tproxy.Debug("FTP: CWD /")
	err2 := conn.ChangeDir("/")
	if err2 != nil {
		ftpp.tproxy.Debug("FTP: CWD: %s", err2)
		conn.Close()
	}

	return files, 0, err
}

//
// Send a error response
//
func (ftpp *FTPProxy) sendError(w http.ResponseWriter, httpStatus int, err error) {
	if ftperr, ok := err.(*textproto.Error); ok {
		err = fmt.Errorf("FTP: %s", err)

		if httpStatus == 0 {
			switch ftperr.Code {
			case ftp.StatusNotLoggedIn:
				httpStatus = http.StatusUnauthorized
			}
		}
	}

	ftpp.tproxy.httpError(w, httpStatus, err)
}

//
// Send a response with directory listing
//
func (ftpp *FTPProxy) sendDirectory(w http.ResponseWriter, r *http.Request,
	path string, files []*ftp.Entry) {

	// Prepare list of files
	sort.Slice(files, func(i, j int) bool {
		f1 := files[i]
		f2 := files[j]

		// Directories first
		switch {
		case f1.Type == ftp.EntryTypeFolder && f2.Type != ftp.EntryTypeFolder:
			return true
		case f1.Type != ftp.EntryTypeFolder && f2.Type == ftp.EntryTypeFolder:
			return false
		}

		// Special folders first
		switch {
		case f1.Name == "." && f2.Name != ".":
			return true
		case f1.Name != "." && f2.Name == ".":
			return false
		case f1.Name == ".." && f2.Name != "..":
			return true
		case f1.Name != ".." && f2.Name == "..":
			return false
		}

		// Then sort by name
		return f1.Name < f2.Name
	})

	// Make sure we have parent directory
	switch {
	case len(files) > 0 && files[0].Name == "..":
	case len(files) > 1 && files[1].Name == "..":
	default:
		files = append([]*ftp.Entry{{Name: "..", Type: ftp.EntryTypeFolder}}, files...)
	}

	// Format HTML head
	w.Write([]byte("<html>"))

	w.Write([]byte(`<head><meta charset="utf-8">` + "\n"))
	w.Write([]byte("<style>\n"))
	w.Write([]byte("th, td {\n"))
	w.Write([]byte("    padding-right: 15px;\n"))
	w.Write([]byte("}\n"))
	w.Write([]byte("</style>\n"))
	w.Write([]byte("</head>\n"))

	w.Write([]byte("<title>"))
	template.HTMLEscape(w, []byte(r.URL.String()))
	w.Write([]byte("</title>\n"))
	w.Write([]byte("<body>\n"))

	// Format table of files
	w.Write([]byte(`<fieldset style="border-radius:10px">`))
	fmt.Fprintf(w, "<legend>Listing of %s</legend>\n", template.HTMLEscapeString(path))
	w.Write([]byte("<table><tbody>\n"))

	for _, f := range files {
		var href, name, symbol string

		switch f.Name {
		case ".":
			continue

		case "..":
			href = path
			i := 0
			switch i = strings.LastIndexByte(href, '/'); {
			case i > 0:
				href = href[:i] + "/"
			case i == 0:
				href = "/"
			}
			name = "Parent directory"
			symbol = "&#x1f8a0;"

		default:
			href = path
			if len(href) > 1 {
				href += "/"
			}
			href += f.Name

			name = template.HTMLEscapeString(f.Name)

			switch f.Type {
			case ftp.EntryTypeFolder:
				symbol = "&#x1f4c2;"
				href += "/"
			default:
				symbol = "&#x1f4c4;"
			}
		}

		// Format file time and size
		time := ""
		size := ""
		if f.Type != ftp.EntryTypeFolder {
			switch {
			case f.Size < 1024:
				size = fmt.Sprintf("%d", f.Size)
			case f.Size < 1024*1024:
				size = fmt.Sprintf("%.1fK", float64(f.Size)/1024)
			case f.Size < 1024*1024*1024:
				size = fmt.Sprintf("%.1fM", float64(f.Size)/(1024*1024))
			case f.Size < 1024*1024*1024*1024:
				size = fmt.Sprintf("%.1fG", float64(f.Size)/(1024*1024*1024))
			}

			time = fmt.Sprintf("%.2d-%.2d-%.4d %.2d:%.2d",
				f.Time.Day(),
				f.Time.Month(),
				f.Time.Year(),
				f.Time.Hour(),
				f.Time.Minute(),
			)
		}

		// Create table row
		w.Write([]byte("<tr>"))
		fmt.Fprintf(w, `<td>%s&nbsp;<a href=%q>%s</a></td>`, symbol, href, name)
		fmt.Fprintf(w, `<td>%s</td>`, size)
		fmt.Fprintf(w, `<td>%s</td>`, time)
		w.Write([]byte("</tr>\n"))
	}

	w.Write([]byte("</tbody></table>\n"))
	w.Write([]byte("</fieldset></body></html>\n"))
}

//
// Send a response with directory listing
//
func (ftpp *FTPProxy) sendFile(w http.ResponseWriter, body *ftp.Response) {
	io.Copy(w, body)
	body.Close()
}

//
// Dial a connection
//
func (ftpp *FTPProxy) dialConn(transport Transport, site_url url.URL) (*ftpConn, error) {
	// Connect
	addr := NetDefaultPort(site_url.Host, "21")
	netconn, err := transport.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	// create ftpConn
	conn := &ftpConn{ftpp: ftpp, netconn: netconn, site: site_url.String()}

	ftpp.tproxy.Debug("FTP: trying %s", addr)
	conn.ServerConn, err = ftp.DialWithOptions(addr, ftp.DialWithNetConn(netconn))
	if err != nil {
		ftpp.tproxy.Debug("FTP: %s: %s", err)
		netconn.Close()
		return nil, err
	}

	// Login
	user := site_url.User.Username()
	pass, _ := site_url.User.Password()
	if user == "" {
		user, pass = "anonymous", "anonymous"
	}

	ftpp.tproxy.Debug("FTP: login %s %s", user, pass)
	err = conn.Login(user, pass)
	if err != nil {
		ftpp.tproxy.Debug("FTP: login %s %s: %s", user, pass, err)
		netconn.Close()
		return nil, err
	} else {
		ftpp.tproxy.Debug("FTP: login %s %s: OK", user, pass)
	}

	ftpp.tproxy.IncCounter(&ftpp.tproxy.Counters.FTPConnections)

	return conn, nil
}

//
// Get a connection
//
func (ftpp *FTPProxy) getConn(site string) *ftpConn {
	ftpp.idleLock.Lock()
	defer ftpp.idleLock.Unlock()

	var conn *ftpConn
	if bucket := ftpp.idleConnections[site]; bucket != nil {
		var expires time.Time
		for c, t := range bucket {
			if conn == nil || expires.After(t) {
				conn, expires = c, t
			}
		}

		if conn != nil {
			delete(bucket, conn)
			conn.reused = true
		}

		if len(bucket) == 0 {
			delete(ftpp.idleConnections, site)
		}
	}

	return conn
}

//
// Put a connection
//
func (conn *ftpConn) putConn() {
	if conn.closed {
		return
	}

	conn.ftpp.idleLock.Lock()
	defer conn.ftpp.idleLock.Unlock()

	bucket := conn.ftpp.idleConnections[conn.site]
	if bucket == nil {
		bucket = make(ftpIddleConnBucket)
		conn.ftpp.idleConnections[conn.site] = bucket
	}

	if _, found := bucket[conn]; found {
		panic("internal error")
	}

	bucket[conn] = time.Now().Add(5 * time.Minute)

	select {
	case conn.ftpp.idleChan <- struct{}{}:
	}
}

//
// Close a connection
//
func (conn *ftpConn) Close() {
	if !conn.closed {
		conn.closed = true
		conn.netconn.Close()
		conn.ftpp.tproxy.DecCounter(&conn.ftpp.tproxy.Counters.FTPConnections)
	}
}

//
// This function expires idle connections. It runs as a goroutine
//
func (ftpp *FTPProxy) expireIdleConnections() {
	timer := time.NewTimer(time.Hour)
	timer.Stop()

	for {
		select {
		case <-timer.C:
		case _, ok := <-ftpp.idleChan:
			if !ok {
				return
			}
		}

		now := time.Now()
		next := now.Add(1000 * time.Hour)

		for site, bucket := range ftpp.idleConnections {
			for conn, exp := range bucket {
				switch {
				case !exp.After(now):
					conn.Close()
					delete(bucket, conn)
				case exp.Before(next):
					next = exp
				}
			}

			if len(bucket) == 0 {
				delete(ftpp.idleConnections, site)
			}
		}

		if len(ftpp.idleConnections) != 0 {
			timer.Reset(next.Sub(now))
		}
	}
}
