// Package logzap adapts a *zap.Logger to the gotd log.Logger port.
//
// It lives in its own module so that go.uber.org/zap never enters the core
// github.com/gotd/log dependency graph.
package logzap

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gotd/log"
)

// New returns a log.Logger that writes to l.
func New(l *zap.Logger) log.Logger {
	return logger{l: l}
}

type logger struct {
	l *zap.Logger
}

func (g logger) Enabled(_ context.Context, level log.Level) bool {
	return g.l.Core().Enabled(zapLevel(level))
}

func (g logger) Log(_ context.Context, level log.Level, msg string, attrs ...log.Attr) {
	ce := g.l.Check(zapLevel(level), msg)
	if ce == nil {
		return
	}

	fields := make([]zap.Field, len(attrs))
	for i, a := range attrs {
		fields[i] = zapField(a)
	}

	ce.Write(fields...)
}

func zapLevel(l log.Level) zapcore.Level {
	switch {
	case l < log.LevelInfo:
		return zapcore.DebugLevel
	case l < log.LevelWarn:
		return zapcore.InfoLevel
	case l < log.LevelError:
		return zapcore.WarnLevel
	default:
		return zapcore.ErrorLevel
	}
}

func zapField(a log.Attr) zap.Field {
	v := a.Value
	switch v.Kind() {
	case log.KindString:
		return zap.String(a.Key, v.String())
	case log.KindInt64:
		return zap.Int64(a.Key, v.Int64())
	case log.KindUint64:
		return zap.Uint64(a.Key, v.Uint64())
	case log.KindFloat64:
		return zap.Float64(a.Key, v.Float64())
	case log.KindBool:
		return zap.Bool(a.Key, v.Bool())
	case log.KindDuration:
		return zap.Duration(a.Key, v.Duration())
	case log.KindTime:
		return zap.Time(a.Key, v.Time())
	case log.KindError:
		return zap.NamedError(a.Key, v.Error())
	default:
		return zap.Any(a.Key, v.Any())
	}
}
