package loglogrus_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gotd/log"
	"github.com/gotd/log/loglogrus"
)

func newLogger(buf *bytes.Buffer, level logrus.Level) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(buf)
	l.SetFormatter(&logrus.JSONFormatter{})
	l.SetLevel(level)
	return l
}

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
	l := loglogrus.New(newLogger(&buf, logrus.DebugLevel))

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
	if rec["msg"] != "hello" {
		t.Errorf("msg = %v", rec["msg"])
	}
	if rec["level"] != "warning" {
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
	l := loglogrus.New(newLogger(&buf, logrus.DebugLevel))

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

func TestWithDoesNotMutateParent(t *testing.T) {
	var buf bytes.Buffer
	base := loglogrus.New(newLogger(&buf, logrus.DebugLevel))
	base = log.With(base, log.String("a", "1"))

	child := log.With(base, log.String("b", "2"))
	child.Log(context.Background(), log.LevelInfo, "m")

	// Parent must not have leaked the child's field.
	buf.Reset()
	base.Log(context.Background(), log.LevelInfo, "m")
	rec := decode(t, &buf)
	if _, ok := rec["b"]; ok {
		t.Errorf("child field leaked into parent: %#v", rec)
	}
}

func TestAdapterLevelGate(t *testing.T) {
	var buf bytes.Buffer
	l := loglogrus.New(newLogger(&buf, logrus.WarnLevel))

	ctx := context.Background()
	if l.Enabled(ctx, log.LevelInfo) {
		t.Fatal("info must be gated out at warn level")
	}

	l.Log(ctx, log.LevelInfo, "dropped")
	if buf.Len() != 0 {
		t.Fatalf("info should be gated, got %q", buf.String())
	}
}
