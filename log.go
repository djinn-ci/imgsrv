package main

import (
	"io"
	"log"
	"strings"
)

type LogLevel uint8

//go:generate stringer -type LogLevel -linecomment
const (
	Debug LogLevel = iota + 1 // DEBUG
	Info                      // INFO
	Warn                      // WARN
	Error                     // ERROR
)

var LogLevels = map[string]LogLevel{
	"debug": Debug,
	"info":  Info,
	"warn":  Warn,
	"error": Error,
}

// Logger is the type for logging information at different levels of severity.
// The Logger has three logStates representing each level that can be logged at,
// Debug, Info, and Error.
type Logger struct {
	closer io.Closer

	Debug logState
	Info  logState
	Warn  logState
	Error logState
}

// Queue is a logger that can be given to CurlyQ for logging information about
// the jobs being processed.
type Queue struct {
	*Logger
}

type logState struct {
	logger *log.Logger
	level  LogLevel
	actual LogLevel
}

// NewLog returns a new Logger that will write to the given io.Writer. This will
// use the stdlib's logger with the log.Ldate, log.Ltime, and log.LUTC flags
// set. The default level of the returned Logger is info.
func NewLog(wc io.WriteCloser) *Logger {
	defaultLevel := Info
	logger := log.New(wc, "", log.Ldate|log.Ltime|log.LUTC)

	return &Logger{
		closer: wc,
		Debug: logState{
			logger: logger,
			level:  defaultLevel,
			actual: Debug,
		},
		Info: logState{
			logger: logger,
			level:  defaultLevel,
			actual: Info,
		},
		Warn: logState{
			logger: logger,
			level:  defaultLevel,
			actual: Warn,
		},
		Error: logState{
			logger: logger,
			level:  defaultLevel,
			actual: Error,
		},
	}
}

// SetLevel sets the level of the logger. The level should be either "debug",
// "info", or "error". If the given string is none of these values then the
// logger's level will be unchanged.
func (l *Logger) SetLevel(s string) {
	if lvl, ok := LogLevels[strings.ToLower(s)]; ok {
		l.Debug.level = lvl
		l.Info.level = lvl
		l.Error.level = lvl
	}
}

// SetWriter set's the io.Writer for the underlying logger.
func (l *Logger) SetWriter(w io.WriteCloser) {
	logger := log.New(w, "", log.Ldate|log.Ltime|log.LUTC)

	l.closer = w
	l.Debug.logger = logger
	l.Info.logger = logger
	l.Warn.logger = logger
	l.Error.logger = logger
}

func (l *Logger) Close() error { return l.closer.Close() }

func (s *logState) Printf(format string, v ...interface{}) {
	if s.actual < s.level {
		return
	}
	s.logger.Printf(s.actual.String()+" "+format, v...)
}

func (s *logState) Println(v ...interface{}) {
	if s.actual < s.level {
		return
	}
	s.logger.Println(append([]interface{}{s.actual}, v...)...)
}

func (s *logState) Fatalf(format string, v ...interface{}) {
	s.logger.Fatalf(s.actual.String()+" "+format, v...)
}

func (s *logState) Fatal(v ...interface{}) {
	s.logger.Fatal(append([]interface{}{s.actual, " "}, v...)...)
}
