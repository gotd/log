package log

import "context"

// Helper wraps a Logger with leveled convenience methods for call sites. It is a
// thin value type; adapters implement Logger, not Helper.
//
// Build one with For; the zero Helper logs to Nop.
type Helper struct {
	l Logger
}

// For returns a Helper writing to l, or to Nop if l is nil.
func For(l Logger) Helper {
	return Helper{l: OrNop(l)}
}

func (h Helper) logger() Logger {
	if h.l == nil {
		return Nop
	}
	return h.l
}

// Enabled reports whether a record at level would be recorded.
func (h Helper) Enabled(ctx context.Context, level Level) bool {
	return h.logger().Enabled(ctx, level)
}

// Debug logs at LevelDebug.
func (h Helper) Debug(ctx context.Context, msg string, attrs ...Attr) {
	h.logger().Log(ctx, LevelDebug, msg, attrs...)
}

// Info logs at LevelInfo.
func (h Helper) Info(ctx context.Context, msg string, attrs ...Attr) {
	h.logger().Log(ctx, LevelInfo, msg, attrs...)
}

// Warn logs at LevelWarn.
func (h Helper) Warn(ctx context.Context, msg string, attrs ...Attr) {
	h.logger().Log(ctx, LevelWarn, msg, attrs...)
}

// Error logs at LevelError.
func (h Helper) Error(ctx context.Context, msg string, attrs ...Attr) {
	h.logger().Log(ctx, LevelError, msg, attrs...)
}
