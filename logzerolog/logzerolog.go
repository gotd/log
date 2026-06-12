// Package logzerolog adapts a zerolog.Logger to the gotd log.Logger port.
//
// It lives in its own module so that github.com/rs/zerolog never enters the
// core github.com/gotd/log dependency graph.
package logzerolog

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/gotd/log"
)

// New returns a log.Logger that writes to l.
//
// zerolog.Logger is a value type, so pass it by value (e.g. zerolog.New(w) or
// log.Logger from the zerolog package).
func New(l zerolog.Logger) log.Logger {
	return logger{l: l}
}

type logger struct {
	l zerolog.Logger
}

func (g logger) Enabled(_ context.Context, level log.Level) bool {
	zl := zerologLevel(level)
	return zl >= g.l.GetLevel() && zl >= zerolog.GlobalLevel()
}

func (g logger) Log(_ context.Context, level log.Level, msg string, attrs ...log.Attr) {
	// WithLevel returns a disabled event when the level is gated out; adding
	// fields and calling Msg on it are no-ops.
	e := g.l.WithLevel(zerologLevel(level))
	for _, a := range attrs {
		e = addEvent(e, a)
	}
	e.Msg(msg)
}

// With implements log.Wither, mapping onto zerolog's context builder.
func (g logger) With(attrs ...log.Attr) log.Logger {
	c := g.l.With()
	for _, a := range attrs {
		c = addContext(c, a)
	}
	return logger{l: c.Logger()}
}

func zerologLevel(l log.Level) zerolog.Level {
	switch {
	case l < log.LevelInfo:
		return zerolog.DebugLevel
	case l < log.LevelWarn:
		return zerolog.InfoLevel
	case l < log.LevelError:
		return zerolog.WarnLevel
	default:
		return zerolog.ErrorLevel
	}
}

// addEvent attaches a to e and returns the event for chaining.
func addEvent(e *zerolog.Event, a log.Attr) *zerolog.Event {
	v := a.Value
	switch v.Kind() {
	case log.KindString:
		return e.Str(a.Key, v.String())
	case log.KindInt64:
		return e.Int64(a.Key, v.Int64())
	case log.KindUint64:
		return e.Uint64(a.Key, v.Uint64())
	case log.KindFloat64:
		return e.Float64(a.Key, v.Float64())
	case log.KindBool:
		return e.Bool(a.Key, v.Bool())
	case log.KindDuration:
		return e.Dur(a.Key, v.Duration())
	case log.KindTime:
		return e.Time(a.Key, v.Time())
	case log.KindError:
		return e.AnErr(a.Key, v.Error())
	case log.KindGroup:
		children := v.Group()
		// An empty key inlines the children into the parent (see log.Group).
		if a.Key == "" {
			for _, c := range children {
				e = addEvent(e, c)
			}
			return e
		}
		d := zerolog.Dict()
		for _, c := range children {
			d = addEvent(d, c)
		}
		return e.Dict(a.Key, d)
	default:
		return e.Interface(a.Key, v.Any())
	}
}

// addContext attaches a to c and returns the context for chaining. It mirrors
// addEvent for the child-logger (With) path.
func addContext(c zerolog.Context, a log.Attr) zerolog.Context {
	v := a.Value
	switch v.Kind() {
	case log.KindString:
		return c.Str(a.Key, v.String())
	case log.KindInt64:
		return c.Int64(a.Key, v.Int64())
	case log.KindUint64:
		return c.Uint64(a.Key, v.Uint64())
	case log.KindFloat64:
		return c.Float64(a.Key, v.Float64())
	case log.KindBool:
		return c.Bool(a.Key, v.Bool())
	case log.KindDuration:
		return c.Dur(a.Key, v.Duration())
	case log.KindTime:
		return c.Time(a.Key, v.Time())
	case log.KindError:
		return c.AnErr(a.Key, v.Error())
	case log.KindGroup:
		children := v.Group()
		if a.Key == "" {
			for _, ch := range children {
				c = addContext(c, ch)
			}
			return c
		}
		d := zerolog.Dict()
		for _, ch := range children {
			d = addEvent(d, ch)
		}
		return c.Dict(a.Key, d)
	default:
		return c.Interface(a.Key, v.Any())
	}
}
