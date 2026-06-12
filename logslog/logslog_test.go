package logslog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/gotd/log"
	"github.com/gotd/log/logslog"
)

func TestAdapter(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := logslog.New(slog.New(h))

	ctx := context.Background()
	if !l.Enabled(ctx, log.LevelDebug) {
		t.Fatal("debug must be enabled at LevelDebug")
	}

	l.Log(ctx, log.LevelInfo, "hello",
		log.String("s", "v"),
		log.Int("n", 7),
		log.Bool("ok", true),
		log.Duration("d", time.Second),
	)

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rec["msg"] != "hello" {
		t.Errorf("msg = %v", rec["msg"])
	}
	if rec["level"] != "INFO" {
		t.Errorf("level = %v", rec["level"])
	}
	if rec["s"] != "v" {
		t.Errorf("s = %v", rec["s"])
	}
	if rec["n"] != float64(7) {
		t.Errorf("n = %v", rec["n"])
	}
	if rec["ok"] != true {
		t.Errorf("ok = %v", rec["ok"])
	}
}

func TestAdapterLevelGate(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	l := logslog.New(slog.New(h))

	ctx := context.Background()
	if l.Enabled(ctx, log.LevelInfo) {
		t.Fatal("info must be disabled when handler level is warn")
	}

	l.Log(ctx, log.LevelInfo, "dropped")
	if buf.Len() != 0 {
		t.Fatalf("info record should be gated out, got %q", buf.String())
	}
}

func TestNewNilUsesDefault(t *testing.T) {
	if logslog.New(nil) == nil {
		t.Fatal("New(nil) must return a usable logger")
	}
}

func TestAdapterCaller(t *testing.T) {
	newLogger := func(buf *bytes.Buffer) *slog.Logger {
		h := slog.NewTextHandler(buf, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug})
		return slog.New(h)
	}
	ctx := context.Background()

	t.Run("Direct", func(t *testing.T) {
		var buf bytes.Buffer
		l := logslog.New(newLogger(&buf))
		l.Log(ctx, log.LevelInfo, "m") // caller line
		assertCaller(t, buf.String())
	})

	t.Run("Helper", func(t *testing.T) {
		var buf bytes.Buffer
		h := log.For(logslog.New(newLogger(&buf)))
		h.Info(ctx, "m") // caller line
		assertCaller(t, buf.String())
	})
}

func assertCaller(t *testing.T, out string) {
	t.Helper()
	if strings.Contains(out, "logslog.go:") {
		t.Errorf("source points at adapter, not caller: %q", out)
	}
	if !strings.Contains(out, "logslog_test.go:") {
		t.Errorf("source = %q, want logslog_test.go", out)
	}
}
