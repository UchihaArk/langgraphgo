package log

import (
	"github.com/kataras/golog"
)

// GologLogger implements Logger interface using kataras/golog
type GologLogger struct {
	logger *golog.Logger
	level  LogLevel
}

var _ Logger = (*GologLogger)(nil)

// NewGologLogger creates a new logger using an existing golog.Logger
func NewGologLogger(logger *golog.Logger) *GologLogger {
	return &GologLogger{
		logger: logger,
		level:  LogLevelInfo, // default level
	}
}

// Debug logs debug messages
func (l *GologLogger) Debug(format string, v ...any) {
	if l.level <= LogLevelDebug {
		args := append([]any{format}, v...)
		l.logger.Debug(args...)
	}
}

// Info logs informational messages
func (l *GologLogger) Info(format string, v ...any) {
	if l.level <= LogLevelInfo {
		args := append([]any{format}, v...)
		l.logger.Info(args...)
	}
}

// Warn logs warning messages
func (l *GologLogger) Warn(format string, v ...any) {
	if l.level <= LogLevelWarn {
		args := append([]any{format}, v...)
		l.logger.Warn(args...)
	}
}

// Error logs error messages
func (l *GologLogger) Error(format string, v ...any) {
	if l.level <= LogLevelError {
		args := append([]any{format}, v...)
		l.logger.Error(args...)
	}
}

// SetLevel sets the log level
func (l *GologLogger) SetLevel(level LogLevel) {
	l.level = level

	// Convert to golog level string
	gologLevel := "info"
	switch level {
	case LogLevelDebug:
		gologLevel = "debug"
	case LogLevelInfo:
		gologLevel = "info"
	case LogLevelWarn:
		gologLevel = "warn"
	case LogLevelError:
		gologLevel = "error"
	case LogLevelNone:
		gologLevel = "disable"
	}

	l.logger.SetLevel(gologLevel)
}

// GetLevel returns the current log level
func (l *GologLogger) GetLevel() LogLevel {
	return l.level
}
