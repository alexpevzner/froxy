package log_test

import "cikorio/log"
import "testing"

func TestLog(_ *testing.T) {
	var l log.Logger

	l.SetFlags(log.WithClock)

	l.SetPrefix("root")
	l.Trace("hello %s", "trace")
	l.Debug("hello %s", "debug")
	l.Info("hello %s", "info")
	l.Warn("hello %s", "warn")
	l.Error("hello %s", "error")

	c := l.ChildWithPrefix("child")
	c.Trace("hello %s", "trace")
	c.Debug("hello %s", "debug")
	c.Info("hello %s", "info")
	c.Warn("hello %s", "warn")
	c.Error("hello %s", "error")
}

// vim:ts=8:sw=4:et
