package types

import (
	ethlog "github.com/ethereum/go-ethereum/log"

	cmtlog "github.com/cometbft/cometbft/libs/log"
)

var _ CompositeLogger = (*compositeLogger)(nil)

// CompositeLogger is the interface for a logger that implements both the cmtlog.Logger and ethlog.Logger interfaces
type CompositeLogger interface {
	cmtlog.Logger
	ethlog.Logger
}

// compositeLogger is a logger that implements both the cmtlog.Logger and ethlog.Logger interfaces
type compositeLogger struct {
	ethlog.Logger
}

// NewCompositeLogger creates a new CompositeLogger
// Accepted log levels are: "trace", "debug", "info", "warn", "error", "crit"
func NewCompositeLogger(lvl string, keyvals ...any) (CompositeLogger, error) {
	ethLogger := ethlog.New(keyvals...)

	lvlHandler, err := ethlog.LvlFromString(lvl)
	if err != nil {
		return nil, err
	}

	ethLogger.SetHandler(ethlog.LvlFilterHandler(lvlHandler, ethLogger.GetHandler()))

	return &compositeLogger{ethLogger}, nil
}

// New implements the ethlog.Logger interface
func (l *compositeLogger) New(keyvals ...any) ethlog.Logger {
	return &compositeLogger{l.Logger.New(keyvals...)}
}

// GetHandler implements the ethlog.Logger interface
func (l *compositeLogger) GetHandler() ethlog.Handler {
	return l.Logger.GetHandler()
}

// SetHandler implements the ethlog.Logger interface
func (l *compositeLogger) SetHandler(h ethlog.Handler) {
	l.Logger.SetHandler(h)
}

// Trace implements the ethlog.Logger interface
func (l *compositeLogger) Trace(msg string, keyvals ...any) {
	l.Logger.Trace(msg, keyvals...)
}

// Debug implements the ethlog.Logger and cmtlog.Logger interfaces
func (l *compositeLogger) Debug(msg string, keyvals ...any) {
	l.Logger.Debug(msg, keyvals...)
}

// Info implements the ethlog.Logger and cmtlog.Logger interfaces
func (l *compositeLogger) Info(msg string, keyvals ...any) {
	l.Logger.Info(msg, keyvals...)
}

// Warn implements the ethlog.Logger interface
func (l *compositeLogger) Warn(msg string, keyvals ...any) {
	l.Logger.Warn(msg, keyvals...)
}

// Error implements the ethlog.Logger and cmtlog.Logger interfaces
func (l *compositeLogger) Error(msg string, keyvals ...any) {
	l.Logger.Error(msg, keyvals...)
}

// Crit implements the ethlog.Logger interface
func (l *compositeLogger) Crit(msg string, keyvals ...any) {
	l.Logger.Crit(msg, keyvals...)
}

// With implements the cmtlog.Logger interface
func (l *compositeLogger) With(keyvals ...any) cmtlog.Logger {
	return &compositeLogger{l.New(keyvals...)}
}
