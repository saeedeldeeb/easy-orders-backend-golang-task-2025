package logger

import (
	"strings"

	"go.uber.org/zap"
)

// Logger wraps zap.Logger for application use
type Logger struct {
	*zap.SugaredLogger
}

// New creates a new logger with the specified level
func New(level string) (*Logger, error) {
	var zapLevel zap.AtomicLevel

	switch strings.ToLower(level) {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn", "warning":
		zapLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	config := zap.Config{
		Level:       zapLevel,
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{SugaredLogger: logger.Sugar()}, nil
}

// NewDevelopment creates a development logger with a pretty output
func NewDevelopment() (*Logger, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	return &Logger{SugaredLogger: logger.Sugar()}, nil
}

// Fatal logs a message and exits
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.SugaredLogger.Fatalf(msg, args...)
}
