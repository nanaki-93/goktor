package service

import "fmt"

// Logger interface for flexible logging
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// DefaultLogger implements Logger interface using fmt
type DefaultLogger struct{}

func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("ℹ [INFO] %s %v\n", msg, args)
}

func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	fmt.Printf("⚠ [WARN] %s %v\n", msg, args)
}

func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	fmt.Printf("✗ [ERROR] %s %v\n", msg, args)
}

func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	fmt.Printf("🔍 [DEBUG] %s %v\n", msg, args)
}
