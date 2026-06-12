// Package loglogrus adapts a *logrus.Logger to the gotd log.Logger port.
//
// It lives in its own module so that github.com/sirupsen/logrus never enters
// the core github.com/gotd/log dependency graph.
package loglogrus

import (
	"context"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/gotd/log"
)

// New returns a log.Logger that writes to l.
//
// logrus exposes no per-call caller-skip option, so when l has ReportCaller
// enabled it would report this adapter as the caller. New installs a hook that
// recomputes the caller from the stack, reporting the code calling log.Logger
// instead. The hook is a no-op when ReportCaller is disabled.
func New(l *logrus.Logger) log.Logger {
	addCallerHook(l)
	return logger{l: l}
}

const (
	logrusPkg  = "github.com/sirupsen/logrus"
	adapterPkg = "github.com/gotd/log/loglogrus"
	// corePkg is the gotd/log facade. Its wrappers (e.g. Helper) sit between the
	// call site and this adapter, so skip them too. Note corePkg+"." does not
	// match adapterPkg frames, which live under corePkg+"/loglogrus".
	corePkg = "github.com/gotd/log"
)

// callerHook rewrites Entry.Caller to the first frame outside logrus and this
// adapter. logrus sets Caller before firing hooks and formats after, so the
// overwrite takes effect (see logrus Entry.log).
type callerHook struct{}

func (callerHook) Levels() []logrus.Level { return logrus.AllLevels }

func (callerHook) Fire(e *logrus.Entry) error {
	if e.Logger == nil || !e.Logger.ReportCaller {
		return nil
	}
	if f := adapterCaller(); f != nil {
		e.Caller = f
	}
	return nil
}

// addCallerHook installs callerHook on l, skipping if one is already present so
// repeated New calls on the same logger do not stack hooks.
func addCallerHook(l *logrus.Logger) {
	for _, h := range l.Hooks[logrus.PanicLevel] {
		if _, ok := h.(callerHook); ok {
			return
		}
	}
	l.AddHook(callerHook{})
}

// adapterCaller returns the first stack frame outside logrus and this adapter,
// i.e. the application code that called log.Logger.
func adapterCaller() *runtime.Frame {
	pcs := make([]uintptr, 25)
	n := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	for {
		f, more := frames.Next()
		if !strings.HasPrefix(f.Function, logrusPkg+".") &&
			!strings.HasPrefix(f.Function, adapterPkg+".") &&
			!strings.HasPrefix(f.Function, corePkg+".") {
			fr := f
			return &fr
		}
		if !more {
			return nil
		}
	}
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
