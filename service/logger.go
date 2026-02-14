package service

import (
	"fmt"
)

// Logger interface for flexible logging
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

const (
	ErrorLevel = iota
	WarnLevel
	InfoLevel
	DebugLevel
)

// DefaultLogger implements Logger interface using fmt
type DefaultLogger struct {
	level int
}

func NewDefaultLogger() Logger {
	return &DefaultLogger{level: InfoLevel}
}

func NewLogger(debug bool) Logger {

	if debug {
		return &DefaultLogger{level: DebugLevel}
	}

	return &DefaultLogger{level: InfoLevel}
}
func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	if l.level < InfoLevel {
		return
	}
	fmt.Printf("ℹ [INFO] %s %v\n", msg, args)
}

func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	if l.level < WarnLevel {
		return
	}
	fmt.Printf("⚠ [WARN] %s %v\n", msg, args)
}

func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	if l.level < ErrorLevel {
		return
	}
	fmt.Printf("✗ [ERROR] %s %v\n", msg, args)
}

func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	if l.level < DebugLevel {
		return
	}
	fmt.Printf("🔍 [DEBUG] %s %v\n", msg, args)
}
