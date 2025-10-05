// Package logger provides structured logging for Dirvana.
package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus logger
type Logger struct {
	log *logrus.Logger
}

// Entry wraps logrus entry for method chaining
type Entry struct {
	entry *logrus.Entry
}

// New creates a new logger instance
func New(level string, output io.Writer) *Logger {
	if output == nil {
		output = os.Stderr
	}

	log := logrus.New()
	log.SetOutput(output)

	// Parse level
	logLevel, err := logrus.ParseLevel(strings.ToLower(level))
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	log.SetLevel(logLevel)

	// Use simple text formatter with colors
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
		PadLevelText:     true,
	})

	return &Logger{log: log}
}

// Debug logs a debug message
func (l *Logger) Debug() *Entry {
	return &Entry{entry: logrus.NewEntry(l.log).WithField("level", "debug")}
}

// Info logs an info message
func (l *Logger) Info() *Entry {
	return &Entry{entry: logrus.NewEntry(l.log).WithField("level", "info")}
}

// Warn logs a warning message
func (l *Logger) Warn() *Entry {
	return &Entry{entry: logrus.NewEntry(l.log).WithField("level", "warn")}
}

// Error logs an error message
func (l *Logger) Error() *Entry {
	return &Entry{entry: logrus.NewEntry(l.log).WithField("level", "error")}
}

// Str adds a string field
func (e *Entry) Str(key, value string) *Entry {
	e.entry = e.entry.WithField(key, value)
	return e
}

// Int adds an int field
func (e *Entry) Int(key string, value int) *Entry {
	e.entry = e.entry.WithField(key, value)
	return e
}

// Bool adds a bool field
func (e *Entry) Bool(key string, value bool) *Entry {
	e.entry = e.entry.WithField(key, value)
	return e
}

// Err adds an error field
func (e *Entry) Err(err error) *Entry {
	if err != nil {
		e.entry = e.entry.WithError(err)
	}
	return e
}

// Dur adds a duration field (formatted in milliseconds)
func (e *Entry) Dur(key string, duration time.Duration) *Entry {
	// Log duration in milliseconds for readability
	ms := float64(duration.Microseconds()) / 1000.0
	e.entry = e.entry.WithField(key, ms)
	return e
}

// Float adds a float field
func (e *Entry) Float(key string, value float64) *Entry {
	e.entry = e.entry.WithField(key, value)
	return e
}

// Msg logs the message with accumulated fields
func (e *Entry) Msg(msg string) {
	level := e.entry.Data["level"]
	delete(e.entry.Data, "level")

	switch level {
	case "debug":
		e.entry.Debug(msg)
	case "info":
		e.entry.Info(msg)
	case "warn":
		e.entry.Warn(msg)
	case "error":
		e.entry.Error(msg)
	default:
		e.entry.Info(msg)
	}
}
