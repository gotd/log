package logzerolog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/gotd/log"
	"github.com/gotd/log/logzerolog"
)

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("unmarshal %q: %v", buf.String(), err)
	}
	return rec
}

func TestAdapter(t *testing.T) {
	var buf bytes.Buffer
	zl := zerolog.New(&buf).Level(zerolog.DebugLevel)
	l := logzerolog.New(zl)

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

	rec := decode(t, &buf)
	if rec["message"] != "hello" {
		t.Errorf("message = %v", rec["message"])
	}
	if rec["level"] != "warn" {
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

func TestAdapterWithGroup(t *testing.T) {
	var buf bytes.Buffer
	l := logzerolog.New(zerolog.New(&buf).Level(zerolog.DebugLevel))

	l = log.With(l, log.String("base", "1"))

	l.Log(context.Background(), log.LevelInfo, "m",
		log.Group("obj", log.Int("x", 2)),
		log.Group("", log.String("inlined", "y")),
	)

	rec := decode(t, &buf)
	if rec["base"] != "1" {
		t.Errorf("base = %v", rec["base"])
	}
	obj, ok := rec["obj"].(map[string]any)
	if !ok || obj["x"] != float64(2) {
		t.Errorf("obj = %#v", rec["obj"])
	}
	if rec["inlined"] != "y" {
		t.Errorf("inlined field missing: %#v", rec)
	}
}

func TestAdapterLevelGate(t *testing.T) {
	var buf bytes.Buffer
	l := logzerolog.New(zerolog.New(&buf).Level(zerolog.WarnLevel))

	ctx := context.Background()
	if l.Enabled(ctx, log.LevelInfo) {
		t.Fatal("info must be gated out at warn level")
	}

	l.Log(ctx, log.LevelInfo, "dropped")
	if buf.Len() != 0 {
		t.Fatalf("info should be gated, got %q", buf.String())
	}
}

func TestAdapterCaller(t *testing.T) {
	var buf bytes.Buffer
	zl := zerolog.New(&buf).With().Caller().Logger()
	l := logzerolog.New(zl)

	l.Log(context.Background(), log.LevelInfo, "m") // caller line

	rec := decode(t, &buf)
	got, _ := rec["caller"].(string)
	if strings.Contains(got, "logzerolog.go") {
		t.Errorf("caller = %q, want test file not adapter", got)
	}
	if !strings.Contains(got, "logzerolog_test.go") {
		t.Errorf("caller = %q, want logzerolog_test.go", got)
	}
}

func TestAdapterCallerHelper(t *testing.T) {
	var buf bytes.Buffer
	zl := zerolog.New(&buf).With().Caller().Logger()
	h := log.For(logzerolog.New(zl))

	h.Info(context.Background(), "m") // caller line

	rec := decode(t, &buf)
	got, _ := rec["caller"].(string)
	if strings.Contains(got, "logzerolog.go") || strings.Contains(got, "helper.go") {
		t.Errorf("caller = %q, want test file not adapter/helper", got)
	}
	if !strings.Contains(got, "logzerolog_test.go") {
		t.Errorf("caller = %q, want logzerolog_test.go", got)
	}
}
