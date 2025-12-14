package log

import (
	"testing"

	"github.com/kataras/golog"
	"github.com/stretchr/testify/assert"
)

func TestNewGologLogger(t *testing.T) {
	// Create a golog logger
	glogger := golog.New()

	// Create our GologLogger
	logger := NewGologLogger(glogger)

	assert.NotNil(t, logger)
	assert.Equal(t, LogLevelInfo, logger.GetLevel())
}

func TestGologLogger_LevelControl(t *testing.T) {
	glogger := golog.New()
	logger := NewGologLogger(glogger)

	// Test setting different levels
	logger.SetLevel(LogLevelDebug)
	assert.Equal(t, LogLevelDebug, logger.GetLevel())

	logger.SetLevel(LogLevelError)
	assert.Equal(t, LogLevelError, logger.GetLevel())

	logger.SetLevel(LogLevelNone)
	assert.Equal(t, LogLevelNone, logger.GetLevel())
}

func TestGologLogger_Logging(t *testing.T) {
	glogger := golog.New()
	logger := NewGologLogger(glogger)

	// Set to debug level to ensure all messages are logged
	logger.SetLevel(LogLevelDebug)

	// Test logging methods - these should not panic
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warning message")
	logger.Error("Error message")

	// Test with formatted messages
	logger.Debug("Debug: %s", "test")
	logger.Info("Info: %d", 42)
	logger.Warn("Warn: %v", map[string]string{"key": "value"})
	logger.Error("Error: %f", 3.14)
}

func TestGologLogger_LevelFiltering(t *testing.T) {
	glogger := golog.New()
	logger := NewGologLogger(glogger)

	// Set to error level
	logger.SetLevel(LogLevelError)
	assert.Equal(t, LogLevelError, logger.GetLevel())

	// These methods will check level but won't panic
	logger.Debug("This should be filtered")
	logger.Info("This should be filtered")
	logger.Warn("This should be filtered")
	logger.Error("This should be logged")
}

func TestGologLogger_Implementation(t *testing.T) {
	// Verify GologLogger implements Logger interface
	var _ Logger = (*GologLogger)(nil)

	glogger := golog.New()
	logger := NewGologLogger(glogger)

	assert.NotNil(t, logger)
}

func TestGologLogger_CustomGologInstance(t *testing.T) {
	// Create a custom golog with specific configuration
	glogger := golog.New()
	glogger.SetLevel("error")
	glogger.SetPrefix("[CUSTOM] ")

	logger := NewGologLogger(glogger)
	assert.NotNil(t, logger)

	// Test that our level control works independently
	logger.SetLevel(LogLevelDebug)
	assert.Equal(t, LogLevelDebug, logger.GetLevel())
}
