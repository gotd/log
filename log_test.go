package log_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gotd/log"
)

func TestNop(t *testing.T) {
	if log.Nop.Enabled(context.Background(), log.LevelError) {
		t.Fatal("Nop must never be enabled")
	}
	// Must not panic.
	log.Nop.Log(context.Background(), log.LevelError, "x", log.Int("n", 1))
}

func TestOrNop(t *testing.T) {
	if log.OrNop(nil) != log.Nop {
		t.Fatal("OrNop(nil) must be Nop")
	}
	l := capture{}
	if log.OrNop(&l) != &l {
		t.Fatal("OrNop must return the given logger")
	}
}

func TestLevelString(t *testing.T) {
	for _, tt := range []struct {
		lvl  log.Level
		want string
	}{
		{log.LevelDebug, "DEBUG"},
		{log.LevelDebug + 1, "DEBUG"},
		{log.LevelInfo, "INFO"},
		{log.LevelWarn, "WARN"},
		{log.LevelError, "ERROR"},
		{log.LevelError + 4, "ERROR"},
	} {
		if got := tt.lvl.String(); got != tt.want {
			t.Errorf("Level(%d).String() = %q, want %q", tt.lvl, got, tt.want)
		}
	}
}

func TestHelper(t *testing.T) {
	c := &capture{}
	h := log.For(c)

	h.Debug(context.Background(), "d")
	h.Info(context.Background(), "i")
	h.Warn(context.Background(), "w")
	h.Error(context.Background(), "e")

	want := []log.Level{log.LevelDebug, log.LevelInfo, log.LevelWarn, log.LevelError}
	if len(c.records) != len(want) {
		t.Fatalf("got %d records, want %d", len(c.records), len(want))
	}
	for i, r := range c.records {
		if r.level != want[i] {
			t.Errorf("record %d level = %v, want %v", i, r.level, want[i])
		}
	}
}

func TestHelperZeroValueLogsToNop(t *testing.T) {
	var h log.Helper
	// Must not panic with a nil underlying logger.
	h.Info(context.Background(), "ok")
	if h.Enabled(context.Background(), log.LevelError) {
		t.Fatal("zero Helper must not be enabled")
	}
}

func TestValueRoundTrip(t *testing.T) {
	now := time.Date(2026, 6, 12, 1, 2, 3, 0, time.UTC)
	errBoom := errors.New("boom")

	for _, tt := range []struct {
		attr    log.Attr
		kind    log.Kind
		wantStr string
		wantAny any
	}{
		{log.String("s", "hi"), log.KindString, "hi", "hi"},
		{log.Int("i", -7), log.KindInt64, "-7", int64(-7)},
		{log.Int64("i64", 42), log.KindInt64, "42", int64(42)},
		{log.Uint64("u", 9), log.KindUint64, "9", uint64(9)},
		{log.Bool("b", true), log.KindBool, "true", true},
		{log.Duration("d", 2*time.Second), log.KindDuration, "2s", 2 * time.Second},
		{log.Time("t", now), log.KindTime, now.String(), now},
		{log.Error(errBoom), log.KindError, "boom", error(errBoom)},
		{log.Any("a", []int{1, 2}), log.KindAny, "[1 2]", []int{1, 2}},
	} {
		v := tt.attr.Value
		if v.Kind() != tt.kind {
			t.Errorf("%s: kind = %v, want %v", tt.attr.Key, v.Kind(), tt.kind)
		}
		if got := v.String(); got != tt.wantStr {
			t.Errorf("%s: String() = %q, want %q", tt.attr.Key, got, tt.wantStr)
		}
	}

	// Float formatting is locale-independent.
	if got := log.Float64("f", 1.5).Value.String(); got != "1.5" {
		t.Errorf("Float64 String() = %q, want %q", got, "1.5")
	}
}

func TestValueTypedAccessors(t *testing.T) {
	if got := log.Int64("", 5).Value.Int64(); got != 5 {
		t.Errorf("Int64 = %d", got)
	}
	if got := log.Uint64("", 5).Value.Uint64(); got != 5 {
		t.Errorf("Uint64 = %d", got)
	}
	if got := log.Float64("", 2.5).Value.Float64(); got != 2.5 {
		t.Errorf("Float64 = %v", got)
	}
	if got := log.Bool("", true).Value.Bool(); !got {
		t.Errorf("Bool = %v", got)
	}
	if got := log.Duration("", time.Minute).Value.Duration(); got != time.Minute {
		t.Errorf("Duration = %v", got)
	}
	if got := log.Error(nil).Value.Error(); got != nil {
		t.Errorf("Error(nil) = %v", got)
	}
}

// capture is a test Logger recording every record.
type capture struct {
	records []record
}

type record struct {
	level log.Level
	msg   string
	attrs []log.Attr
}

func (c *capture) Enabled(context.Context, log.Level) bool { return true }

func (c *capture) Log(_ context.Context, level log.Level, msg string, attrs ...log.Attr) {
	c.records = append(c.records, record{level: level, msg: msg, attrs: append([]log.Attr(nil), attrs...)})
}
