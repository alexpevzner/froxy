//
// Logfile implements writing log to files, with size limit
// and automatic rotation
//

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"time"
)

type Logfile struct {
	path         string   // Path to log file
	rpipe, wpipe *os.File // Read/write ends of log pipe
	file         *os.File // Output file on disk
}

//
// Create a Logfile
//
func NewLogfile(path string) (*Logfile, error) {
	l := &Logfile{path: path}
	var err error
	l.rpipe, l.wpipe, err = os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("Can't create pipe: %s", err)
	}

	err = l.reopen()
	if err != nil {
		return nil, err
	}

	go l.goroutine()

	return l, nil
}

//
// Get os-level handle of file to write logs to
//
func (l *Logfile) Fd() uintptr {
	return l.wpipe.Fd()
}

//
// Goroutine gathers stream of log messages and
// writes them to file, performing rotation when needed
func (l *Logfile) goroutine() {
	r := bufio.NewReader(l.rpipe)
	buf := &bytes.Buffer{}

	for {
		// Fetch next line
		line, err := r.ReadBytes('\n')
		if err != nil {
			return
		}

		// Prepend time prefix
		buf.Reset()
		now := time.Now()

		year, month, day := now.Date()
		fmt.Fprintf(buf, "%2.2d-%2.2d-%4.4d ", day, month, year)

		hour, min, sec := now.Clock()
		fmt.Fprintf(buf, "%2.2d:%2.2d:%2.2d", hour, min, sec)

		buf.WriteString(": ")
		buf.Write(line)

		// Write to output file
		l.file.Write(buf.Bytes())
		l.file.Sync()

		stat, err := l.file.Stat()
		if err == nil && stat.Size() >= LOG_MAX_FILE_SIZE {
			l.rotate()
		}
	}
}

//
// [Re]open the output file
//
func (l *Logfile) reopen() error {
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}

	var err error
	l.file, err = os.OpenFile(l.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	return err
}

//
// gzip the log
//
func (l *Logfile) gzip(ipath, opath string) error {
	// Open input file
	ifile, err := os.Open(ipath)
	if err != nil {
		return err
	}

	defer ifile.Close()

	// Open output file
	ofile, err := os.OpenFile(opath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	// gzip ifile->ofile
	w := gzip.NewWriter(ofile)
	_, err = io.Copy(w, ifile)
	err2 := w.Close()
	err3 := ofile.Close()

	switch {
	case err == nil && err2 != nil:
		err = err2
	case err == nil && err3 != nil:
		err = err3
	}

	// Cleanup and exit
	if err != nil {
		os.Remove(opath)
	}

	return err
}

//
// Rotate log files
//
func (l *Logfile) rotate() {
	prevpath := ""
	for i := LOG_MAX_BACKUP_FILES; i >= 0; i-- {
		nextpath := l.path
		if i > 0 {
			nextpath += fmt.Sprintf(".%d.gz", i-1)
		}

		switch i {
		case LOG_MAX_BACKUP_FILES:
			os.Remove(nextpath)
		case 0:
			err := l.gzip(nextpath, prevpath)
			if err == nil {
				l.file.Truncate(0)
			}
		default:
			os.Rename(nextpath, prevpath)
		}

		prevpath = nextpath
	}

}
