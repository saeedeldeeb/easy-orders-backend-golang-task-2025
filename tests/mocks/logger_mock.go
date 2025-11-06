package mocks

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewNoOpLogger creates a no-op logger for testing
// This logger discards all log messages and is useful for tests
// where we don't want to clutter test output with logs
func NewNoOpLogger() *zap.SugaredLogger {
	// Create a no-op core that discards everything
	noOpCore := zapcore.NewNopCore()

	// Create a logger with the no-op core
	logger := zap.New(noOpCore)

	return logger.Sugar()
}
