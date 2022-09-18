// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// Logger, with log file size limit and automatic rotation

package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

//
// Log levels
//
type LogLevel int

const (
	LogLevelTrace LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// ----- The Logger -----
//
// The logger
//
type Logger struct {
	level   LogLevel     // Current log level
	path    string       // Path to log file
	buf     bytes.Buffer // Buffer for incomplete line
	timelen int          // Length of time prefix
	file    *os.File     // Output file on disk
	lock    sync.Mutex   // Access lock
}

//
// Open log file
//
func (l *Logger) LogToFile(path string) error {
	if l.file != nil {
		panic("Internal error: (*Logger) LogToFile() called twice")
	}
	l.path = path
	return l.reopen()
}

//
// Write Trace-level log message
//
func (l *Logger) Trace(format string, args ...interface{}) {
	l.format(LogLevelTrace, format, args...)
}

//
// Write Debug-level log message
//
func (l *Logger) Debug(format string, args ...interface{}) {
	l.format(LogLevelDebug, format, args...)
}

//
// Write Info-level log message
//
func (l *Logger) Info(format string, args ...interface{}) {
	l.format(LogLevelInfo, format, args...)
}

//
// Write Warn-level log message
//
func (l *Logger) Warn(format string, args ...interface{}) {
	l.format(LogLevelWarn, format, args...)
}

//
// Write Error-level log message
//
func (l *Logger) Error(format string, args ...interface{}) {
	l.format(LogLevelError, format, args...)
}

//
// Write Error-level log message and terminate a program
//
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.format(LogLevelError, format, args...)
	os.Exit(1)
}

//
// Format a log line
//
func (l *Logger) format(level LogLevel, format string, args ...interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if level < l.level {
		return
	}

	if l.buf.Len() == 0 {
		l.fmtTime()
		l.timelen = l.buf.Len()
	}

	fmt.Fprintf(&l.buf, format, args...)
	l.flush(level)
}

//
// Format time prefix
//
func (l *Logger) fmtTime() {
	now := time.Now()

	year, month, day := now.Date()
	fmt.Fprintf(&l.buf, "%2.2d-%2.2d-%4.4d ", day, month, year)

	hour, min, sec := now.Clock()
	fmt.Fprintf(&l.buf, "%2.2d:%2.2d:%2.2d", hour, min, sec)

	l.buf.WriteString(": ")
}

//
// Flush a line of log, collected in the buffer
//
func (l *Logger) flush(level LogLevel) {
	l.buf.WriteByte('\n')
	switch {
	case l.file != nil:
		l.file.Write(l.buf.Bytes())
	case level <= LogLevelInfo:
		os.Stdout.Write(l.buf.Bytes()[l.timelen:])
	default:
		os.Stderr.Write(l.buf.Bytes()[l.timelen:])
	}
	l.buf.Reset()

	stat, err := l.file.Stat()
	if err == nil && stat.Size() >= LOG_MAX_FILE_SIZE {
		l.rotate()
	}
}

//
// [Re]open the output file
//
func (l *Logger) reopen() error {
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
func (l *Logger) gzip(ipath, opath string) error {
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
func (l *Logger) rotate() {
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

// ----- LogWriter -----
//
// Log writer - implements io.Writer interface to
// write to log files
//
type LogWriter struct {
	logger *Logger  // Destination logger
	level  LogLevel // Current log level
}

var _ = io.Writer(&LogWriter{})

//
// Create new LogWriter
//
func (l *Logger) NewLogWriter(level LogLevel) *LogWriter {
	return &LogWriter{l, level}
}

//
// Write to log -- implements io.Writer interface
//
func (w *LogWriter) Write(data []byte) (int, error) {
	l := w.logger

	l.lock.Lock()
	defer l.lock.Unlock()

	size := len(data)

	for {
		i := bytes.IndexByte(data, '\n')
		if i < 0 {
			break
		}

		if l.buf.Len() == 0 {
			l.fmtTime()
		}

		l.buf.Write(data[:i])
		data = data[i+1:]

		l.flush(w.level)
	}

	if len(data) > 0 {
		l.fmtTime()
		l.buf.Write(data)
	}

	return size, nil
}
