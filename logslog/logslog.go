// Package logslog adapts a *slog.Logger to the gotd log.Logger port. It depends
// only on the standard library.
package logslog

import (
	"context"
	"log/slog"
	"runtime"
	"time"

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
	// skip is the extra caller frames to drop, set via WithCallerSkip by
	// wrappers such as Helper that add a frame above Log.
	skip int
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

	// Capture the caller ourselves and hand the record to the handler directly,
	// the way slog's own output methods do. Otherwise slog records this adapter
	// as the source. runtime.Callers skip: 0 is Callers, 1 is Log, 2 is its
	// caller; g.skip accounts for wrappers such as Helper.
	var pcs [1]uintptr
	runtime.Callers(2+g.skip, pcs[:])

	r := slog.NewRecord(time.Now(), lvl, msg, pcs[0])
	r.AddAttrs(sa...)
	_ = g.l.Handler().Handle(ctx, r)
}

// WithCallerSkip implements log.CallerSkipper.
func (g logger) WithCallerSkip(skip int) log.Logger {
	return logger{l: g.l, skip: g.skip + skip}
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
