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
	// Skip one frame so the reported caller is the code calling log.Logger,
	// not this adapter's Log method.
	return logger{l: l.WithOptions(zap.AddCallerSkip(1))}
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

	ce.Write(zapFields(attrs)...)
}

// With implements log.Wither, mapping onto zap.Logger.With.
func (g logger) With(attrs ...log.Attr) log.Logger {
	return logger{l: g.l.With(zapFields(attrs)...)}
}

// Named implements log.Namer, mapping onto zap.Logger.Named.
func (g logger) Named(name string) log.Logger {
	return logger{l: g.l.Named(name)}
}

func zapFields(attrs []log.Attr) []zap.Field {
	fields := make([]zap.Field, len(attrs))
	for i, a := range attrs {
		fields[i] = zapField(a)
	}
	return fields
}

// attrObject adapts a group of attrs to a zapcore.ObjectMarshaler so nested
// groups render as objects (or inline fields, see zapField).
type attrObject []log.Attr

func (o attrObject) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, a := range o {
		zapField(a).AddTo(enc)
	}
	return nil
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
	case log.KindGroup:
		// An empty key inlines the children into the parent (see log.Group).
		if a.Key == "" {
			return zap.Inline(attrObject(v.Group()))
		}
		return zap.Object(a.Key, attrObject(v.Group()))
	default:
		return zap.Any(a.Key, v.Any())
	}
}
