//
// The logger
//

package log

import (
	"bytes"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

//----- Log levels -----
//
// Type Level represents a log level enumeration
//
type Level int32

const (
	TRACE Level = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL

	DefaultLevel = TRACE
)

//
// Log level prefixes
//
var levelPrefixes = map[Level]string{
	TRACE: "T",
	DEBUG: "D",
	INFO:  "I",
	WARN:  "W",
	ERROR: "E",
	FATAL: "F",
}

//
// Get level prefix
//
func (l Level) Prefix() string {
	return levelPrefixes[l]
}

//
// Get Min of two log levels
//
func (l Level) Min(l2 Level) Level {
	if l < l2 {
		return l
	} else {
		return l2
	}
}

//
// Logger flags
//
type Flags uint32

const (
	WithLevel        Flags = 1 << iota // Include log level prefix
	WithClock                          // With monotonic clock since program start
	WithCompactClock                   // Compact clock. Implies WithClock
	WithTime                           // With wall-time
	WithDate                           // With Date. Doesn't imply WithTime
	WithMillisecond                    // Millisecond resolution
	WithMicrosecond                    // Microsecond resolution
	initialized                        // Initialized flags always non-zero

	// Convenient combinations
	WithTimeDate = WithTime | WithDate
	DefaultFlags = Flags(0)
)

//----- Logger -----
//
// Type Logger represents logging endpoint
//
type Logger struct {
	prefix string  // Each-line prefix
	flags  Flags   // Logger flags
	indent int     // Indentation
	parent *Logger // Parent logger
	level  Level   // Verbosity level
}

var start = time.Now()

//
// Format Clock prefix
//
func fmtClock(buf *bytes.Buffer, flags Flags) {
	now := time.Now().Sub(start)
	sec := now / time.Second

	if (flags & WithCompactClock) != 0 {
		fmt.Fprintf(buf, "%4.4d", sec)
	} else {
		fmt.Fprintf(buf, "%2.2d:%2.2d:%2.2d",
			(sec/3600)%24,
			(sec/60)%60,
			sec%60)
	}

	now -= sec * time.Second
	if (flags & WithMicrosecond) != 0 {
		fmt.Fprintf(buf, ".%6.6d", now)
	} else if (flags & WithMillisecond) != 0 {
		fmt.Fprintf(buf, ".%3.3d", now/time.Millisecond)
	}
}

//
// Format Time prefix
//
func fmtTime(buf *bytes.Buffer, flags Flags) {
	now := time.Now()

	if (flags & WithDate) != 0 {
		year, month, day := now.Date()
		fmt.Fprintf(buf, "%2.2d-%2.2d-%4.4d", day, month, year)
	}

	if (flags & WithTime) != 0 {
		if (flags & WithDate) != 0 {
			buf.WriteString(" ")
		}

		hour, min, sec := now.Clock()
		fmt.Fprintf(buf, "%2.2d:%2.2d:%2.2d", hour, min, sec)

		if (flags & WithMicrosecond) != 0 {
			fmt.Fprintf(buf, ".%6.6d", now.Nanosecond()/1e3)
		} else if (flags & WithMillisecond) != 0 {
			fmt.Fprintf(buf, ".%3.3d", now.Nanosecond()/1e6)
		}
	}

}

//
// Insert delimiting space between prefix parts
//
func fmtSpace(buf *bytes.Buffer) {
	if buf.Len() != 0 {
		buf.WriteString(" ")
	}
}

//
// Write message to logger
//
func (l *Logger) write(level Level, m string, path ...*Logger) {
	path = append(path, l)

	if l.parent != nil {
		l.parent.write(level, m, path...)
	} else {
		// Collect indentation and flags; honor log level
		indent := 0
		flags := Flags(0)
		hasprefix := false

		for _, l := range path {
			if level < l.GetLevel() {
				return
			}

			indent += l.indent
			flags |= l.GetFlags()

			if l.prefix != "" {
				hasprefix = true
			}
		}

		// Build prefix
		buf := new(bytes.Buffer)

		if (flags & (WithClock | WithCompactClock)) != 0 {
			fmtClock(buf, flags)
		} else if (flags & (WithTime | WithDate)) != 0 {
			fmtTime(buf, flags)
		}

		if (flags & WithLevel) != 0 {
			fmtSpace(buf)
			buf.WriteString(level.Prefix())
			buf.WriteString(":")
		}

		if hasprefix {
			for i := len(path) - 1; i >= 0; i-- {
				if path[i].prefix != "" {
					fmtSpace(buf)
					buf.WriteString(path[i].prefix)
					buf.WriteString(":")
				}
			}
		}

		// Append indentation
		for indent > 0 {
			buf.WriteString(" ")
			indent--
		}

		// Append message and output
		if m != "" {
			fmtSpace(buf)
			buf.WriteString(m)
		}

		buf.WriteString("\n")
		os.Stdout.Write(buf.Bytes())
	}
}

//
// Generic log-output function with formatting
//
func (l *Logger) logf(level Level, s string, v ...interface{}) {
	l.write(level, fmt.Sprintf(s, v...))
}

//
// Output a TRACE-level log message
//
func (l *Logger) Trace(s string, v ...interface{}) {
	l.logf(TRACE, s, v...)
}

//
// Output a DEBUG-level log message
//
func (l *Logger) Debug(s string, v ...interface{}) {
	l.logf(DEBUG, s, v...)
}

//
// Output a INFO-level log message
//
func (l *Logger) Info(s string, v ...interface{}) {
	l.logf(INFO, s, v...)
}

//
// Output a WARN-level log message
//
func (l *Logger) Warn(s string, v ...interface{}) {
	l.logf(WARN, s, v...)
}

//
// Output an ERROR-level log message
//
func (l *Logger) Error(s string, v ...interface{}) {
	l.logf(ERROR, s, v...)
}

//
// Output a FATAL-level log message and exit
//
func (l *Logger) Exit(s string, v ...interface{}) {
	l.logf(FATAL, s, v...)
	os.Exit(1)
}

//
// Output a FATAL-level log message and raise panic()
//
func (l *Logger) Panic(s string, v ...interface{}) {
	l.logf(FATAL, s, v...)
	panic("Oops")
}

//
// Output hex dump of binary data
//
func (l *Logger) Dump(level Level, data []byte) {
	buf := new(bytes.Buffer)

	off := 0
	for len(data) > 0 {
		sz := len(data)
		if sz > 16 {
			sz = 16
		}

		fmt.Fprintf(buf, "%4.4x: ", off)

		for i := 0; i < sz; i++ {
			c := ' '
			switch i {
			case sz - 1:

			case 3, 11:
				c = ':'
			case 7:
				c = '-'
			}
			fmt.Fprintf(buf, "%2.2x%c", data[i], c)
		}

		for i := sz; i < 16; i++ {
			buf.Write([]byte("   "))
		}

		for i := 0; i < sz; i++ {
			c := data[i]
			if ' ' <= c && c <= 0x7f {
				fmt.Fprintf(buf, "%c", c)
			} else {
				buf.Write([]byte("."))
			}

		}

		l.write(level, string(buf.Bytes()))
		buf.Reset()

		data = data[sz:]
		off += sz
	}
}

//
// Set log prefix
//
func (l *Logger) SetPrefix(prefix string) {
	l.prefix = prefix
}

//
// Get log prefix
//
func (l *Logger) GetPrefix() string {
	return l.prefix
}

//
// Set log indentation
//
func (l *Logger) SetIndent(indent int) {
	l.indent = indent
}

//
// Get log indentation
//
func (l *Logger) GetIndent() int {
	return l.indent
}

//
// Set log flags
//
func (l *Logger) SetFlags(flags Flags) {
	l.flags = flags | initialized
}

//
// Get log flags
//
func (l *Logger) GetFlags() Flags {
	atomic.CompareAndSwapUint32((*uint32)(&l.flags), 0, uint32(DefaultFlags))
	return l.flags
}

//
// Set log level
//
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

//
// Get log level
//
func (l *Logger) GetLevel() Level {
	return l.level
}

//
// Create new top-level logger
//
func NewLogger(prefix string) *Logger {
	return DefaultLogger.ChildWithPrefix(prefix)
}

//
// Create child logger
//
func (l *Logger) Child() *Logger {
	return &Logger{
		flags:  initialized,
		parent: l,
	}
}

//
// Create child logger with prefix
//
func (l *Logger) ChildWithPrefix(prefix string) *Logger {
	child := l.Child()
	child.prefix = prefix
	return child
}

//
// Create child logger with indent
//
func (l *Logger) ChildWithIndent(indent int) *Logger {
	child := l.Child()
	child.indent = indent
	return child
}

//----- Default logger -----
var DefaultLogger Logger

//
// Output a TRACE-level log message
//
func Trace(s string, v ...interface{}) {
	DefaultLogger.Trace(s, v...)
}

//
// Output a DEBUG-level log message
//
func Debug(s string, v ...interface{}) {
	DefaultLogger.Debug(s, v...)
}

//
// Output a INFO-level log message
//
func Info(s string, v ...interface{}) {
	DefaultLogger.Info(s, v...)
}

//
// Output a WARN-level log message
//
func Warn(s string, v ...interface{}) {
	DefaultLogger.Warn(s, v...)
}

//
// Output an ERROR-level log message
//
func Error(s string, v ...interface{}) {
	DefaultLogger.Error(s, v...)
}

//
// Output a FATAL-level log message and exit
//
func Exit(s string, v ...interface{}) {
	DefaultLogger.Exit(s, v...)
}

//
// Output a FATAL-level log message and raise panic()
//
func Panic(s string, v ...interface{}) {
	DefaultLogger.Panic(s, v...)
}

//
// Output hex dump of binary data
//
func Dump(level Level, data []byte) {
	DefaultLogger.Dump(level, data)
}
