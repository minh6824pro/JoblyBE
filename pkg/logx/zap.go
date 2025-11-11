package logx

import (
	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
)

// ZapLogger wraps zap.Logger to implement kratos log.Logger interface
type ZapLogger struct {
	logger *zap.Logger
}

// NewZapLogger wraps zap.NewProduction() for kratos
func NewZapLogger() (log.Logger, error) {
	zapLogger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &ZapLogger{logger: zapLogger}, nil
}

// Log implements kratos log.Logger interface
func (l *ZapLogger) Log(level log.Level, keyvals ...interface{}) error {
	if len(keyvals) == 0 {
		return nil
	}
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "MISSING_VALUE")
	}

	// Extract message and caller if exists
	var msg string
	var hasCaller bool
	fields := make([]zap.Field, 0, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			continue
		}

		if key == "msg" || key == "message" {
			if msgStr, ok := keyvals[i+1].(string); ok {
				msg = msgStr
			}
		} else if key == "caller" {
			hasCaller = true
			fields = append(fields, zap.Any(key, keyvals[i+1]))
		} else {
			fields = append(fields, zap.Any(key, keyvals[i+1]))
		}
	}

	logger := l.logger
	if hasCaller {
		logger = l.logger.WithOptions(zap.WithCaller(false))
	}

	switch level {
	case log.LevelDebug:
		logger.Debug(msg, fields...)
	case log.LevelInfo:
		logger.Info(msg, fields...)
	case log.LevelWarn:
		logger.Warn(msg, fields...)
	case log.LevelError:
		logger.Error(msg, fields...)
	case log.LevelFatal:
		logger.Fatal(msg, fields...)
	default:
		logger.Info(msg, fields...)
	}
	return nil
}

// Sync flushes any buffered log entries
func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}

// GetZapLogger returns the underlying zap logger
func (l *ZapLogger) GetZapLogger() *zap.Logger {
	return l.logger
}
