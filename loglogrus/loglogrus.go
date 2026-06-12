// Package loglogrus adapts a *logrus.Logger to the gotd log.Logger port.
//
// It lives in its own module so that github.com/sirupsen/logrus never enters
// the core github.com/gotd/log dependency graph.
package loglogrus

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/gotd/log"
)

// New returns a log.Logger that writes to l.
func New(l *logrus.Logger) log.Logger {
	return logger{l: l}
}

type logger struct {
	l *logrus.Logger
	// fields are attached by With and prepended to every record. logrus has no
	// native child-logger value distinct from an Entry, so we carry the base
	// fields ourselves and merge them at log time.
	fields logrus.Fields
}

func (g logger) Enabled(_ context.Context, level log.Level) bool {
	return g.l.IsLevelEnabled(logrusLevel(level))
}

func (g logger) Log(_ context.Context, level log.Level, msg string, attrs ...log.Attr) {
	lvl := logrusLevel(level)
	if !g.l.IsLevelEnabled(lvl) {
		return
	}
	g.l.WithFields(g.merge(attrs)).Log(lvl, msg)
}

// With implements log.Wither by accumulating fields on a child logger.
func (g logger) With(attrs ...log.Attr) log.Logger {
	return logger{l: g.l, fields: g.merge(attrs)}
}

// merge returns the base fields combined with attrs. The result is always a
// fresh map, so child loggers never mutate their parent's fields.
func (g logger) merge(attrs []log.Attr) logrus.Fields {
	fields := make(logrus.Fields, len(g.fields)+len(attrs))
	for k, v := range g.fields {
		fields[k] = v
	}
	for _, a := range attrs {
		addField(fields, a)
	}
	return fields
}

func logrusLevel(l log.Level) logrus.Level {
	switch {
	case l < log.LevelInfo:
		return logrus.DebugLevel
	case l < log.LevelWarn:
		return logrus.InfoLevel
	case l < log.LevelError:
		return logrus.WarnLevel
	default:
		return logrus.ErrorLevel
	}
}

// addField writes a into fields, recursing for groups.
func addField(fields logrus.Fields, a log.Attr) {
	v := a.Value
	if v.Kind() == log.KindGroup {
		children := v.Group()
		// An empty key inlines the children into the parent (see log.Group).
		if a.Key == "" {
			for _, c := range children {
				addField(fields, c)
			}
			return
		}
		sub := make(logrus.Fields, len(children))
		for _, c := range children {
			addField(sub, c)
		}
		fields[a.Key] = sub
		return
	}
	fields[a.Key] = fieldValue(v)
}

func fieldValue(v log.Value) any {
	switch v.Kind() {
	case log.KindString:
		return v.String()
	case log.KindInt64:
		return v.Int64()
	case log.KindUint64:
		return v.Uint64()
	case log.KindFloat64:
		return v.Float64()
	case log.KindBool:
		return v.Bool()
	case log.KindDuration:
		return v.Duration()
	case log.KindTime:
		return v.Time()
	case log.KindError:
		// logrus formatters render an error value via its Error method.
		return v.Error()
	default:
		return v.Any()
	}
}
