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
//
//
type ftpConn struct {
	*ftp.ServerConn
	netconn net.Conn
	site    string
	refcnt  int
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

	// Obtain a connection
	conn := ftpp.getConn(site)
	var err error
	if conn == nil {
		conn, err = ftpp.dialConn(transport, site_url)
		if err != nil {
			ftpp.tproxy.httpError(w, http.StatusServiceUnavailable, err)
			return
		}
	}

	defer ftpp.putConn(conn)

	// Perform a transaction
	path := r.URL.Path

	// Try to interpret path as file
	if !strings.HasSuffix(path, "/") {
		var body *ftp.Response
		ftpp.tproxy.Debug("FTP: retr %q", path)
		body, err = conn.Retr(path)
		if err == nil {
			ftpp.sendFile(w, conn, body)
			return
		}
	}

	// Try to interpret path as directory
	ftpp.tproxy.Debug("FTP: LIST %q", path)
	if list, err2 := conn.List(path); err2 == nil {
		ftpp.sendDirectory(w, r, conn, path, list)
		return
	} else {
		if err == nil {
			err = err2
		}
	}

	ftpp.sendError(w, conn, err)
}

//
// Send a response with directory listing
//
func (ftpp *FTPProxy) sendDirectory(w http.ResponseWriter, r *http.Request,
	conn *ftpConn, path string, files []*ftp.Entry) {

	// Normalize path
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

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
func (ftpp *FTPProxy) sendFile(w http.ResponseWriter, conn *ftpConn, body *ftp.Response) {
	io.Copy(w, body)
	body.Close()
}

//
// Send FTP error
//
func (ftpp *FTPProxy) sendError(w http.ResponseWriter, conn *ftpConn, err error) {
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
	conn := &ftpConn{netconn: netconn, site: site_url.String(), refcnt: 1}

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
func (ftpp *FTPProxy) putConn(conn *ftpConn) {
	conn.refcnt--
	if conn.refcnt > 0 {
		return
	}

	ftpp.idleLock.Lock()
	defer ftpp.idleLock.Unlock()

	bucket := ftpp.idleConnections[conn.site]
	if bucket == nil {
		bucket = make(ftpIddleConnBucket)
		ftpp.idleConnections[conn.site] = bucket
	}

	if _, found := bucket[conn]; found {
		panic("internal error")
	}

	bucket[conn] = time.Now().Add(5 * time.Minute)

	select {
	case ftpp.idleChan <- struct{}{}:
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
