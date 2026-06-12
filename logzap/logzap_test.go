package logzap_test

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/gotd/log"
	"github.com/gotd/log/logzap"
)

func TestAdapter(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	l := logzap.New(zap.New(core))

	ctx := context.Background()
	if !l.Enabled(ctx, log.LevelDebug) {
		t.Fatal("debug must be enabled")
	}

	l.Log(ctx, log.LevelWarn, "hello",
		log.String("s", "v"),
		log.Int64("n", 7),
		log.Bool("ok", true),
		log.Duration("d", time.Second),
	)

	all := logs.All()
	if len(all) != 1 {
		t.Fatalf("got %d entries, want 1", len(all))
	}
	e := all[0]
	if e.Message != "hello" {
		t.Errorf("message = %q", e.Message)
	}
	if e.Level != zapcore.WarnLevel {
		t.Errorf("level = %v, want warn", e.Level)
	}

	m := e.ContextMap()
	if m["s"] != "v" {
		t.Errorf("s = %v", m["s"])
	}
	if m["n"] != int64(7) {
		t.Errorf("n = %v", m["n"])
	}
	if m["ok"] != true {
		t.Errorf("ok = %v", m["ok"])
	}
	if m["d"] != time.Second {
		t.Errorf("d = %v", m["d"])
	}
}

func TestAdapterLevelGate(t *testing.T) {
	core, logs := observer.New(zapcore.WarnLevel)
	l := logzap.New(zap.New(core))

	ctx := context.Background()
	if l.Enabled(ctx, log.LevelInfo) {
		t.Fatal("info must be gated out at warn level")
	}

	l.Log(ctx, log.LevelInfo, "dropped")
	if logs.Len() != 0 {
		t.Fatalf("info should be gated, got %d entries", logs.Len())
	}
}
