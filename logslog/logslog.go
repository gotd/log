// Package logslog adapts a *slog.Logger to the gotd log.Logger port. It depends
// only on the standard library.
package logslog

import (
	"context"
	"log/slog"

	"github.com/gotd/log"
)

// New returns a log.Logger that writes to l. If l is nil, slog.Default is used.
func New(l *slog.Logger) log.Logger {
	if l == nil {
		l = slog.Default()
	}
	return logger{l: l}
}

type logger struct {
	l *slog.Logger
}

func (g logger) Enabled(ctx context.Context, level log.Level) bool {
	return g.l.Enabled(ctx, slog.Level(level))
}

func (g logger) Log(ctx context.Context, level log.Level, msg string, attrs ...log.Attr) {
	lvl := slog.Level(level)
	if !g.l.Enabled(ctx, lvl) {
		return
	}

	sa := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		sa[i] = slogAttr(a)
	}

	g.l.LogAttrs(ctx, lvl, msg, sa...)
}

func slogAttr(a log.Attr) slog.Attr {
	v := a.Value
	switch v.Kind() {
	case log.KindString:
		return slog.String(a.Key, v.String())
	case log.KindInt64:
		return slog.Int64(a.Key, v.Int64())
	case log.KindUint64:
		return slog.Uint64(a.Key, v.Uint64())
	case log.KindFloat64:
		return slog.Float64(a.Key, v.Float64())
	case log.KindBool:
		return slog.Bool(a.Key, v.Bool())
	case log.KindDuration:
		return slog.Duration(a.Key, v.Duration())
	case log.KindTime:
		return slog.Time(a.Key, v.Time())
	case log.KindError:
		return slog.Any(a.Key, v.Error())
	case log.KindGroup:
		children := v.Group()
		args := make([]any, len(children))
		for i, c := range children {
			args[i] = slogAttr(c)
		}
		// An empty key inlines the children into the parent, matching slog's
		// own Group semantics and log.Group.
		return slog.Group(a.Key, args...)
	default:
		return slog.Any(a.Key, v.Any())
	}
}
