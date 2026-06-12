package log_test

import (
	"context"
	"testing"

	"github.com/gotd/log"
)

func TestWith(t *testing.T) {
	r := &capture{}
	l := log.With(r, log.String("a", "1"))
	l = log.With(l, log.Int("b", 2))
	l.Log(context.Background(), log.LevelInfo, "msg", log.String("c", "3"))

	if len(r.records) != 1 {
		t.Fatalf("got %d records", len(r.records))
	}
	got := r.records[0].attrs
	if len(got) != 3 {
		t.Fatalf("got %d attrs: %v", len(got), got)
	}
	if got[0].Key != "a" || got[1].Key != "b" || got[2].Key != "c" {
		t.Fatalf("attr order wrong: %v", got)
	}
}

func TestWithDoesNotMutateBase(t *testing.T) {
	r := &capture{}
	base := log.With(r, log.String("a", "1"))
	c1 := log.With(base, log.String("b", "2"))
	c2 := log.With(base, log.String("c", "3"))

	c1.Log(context.Background(), log.LevelInfo, "m")
	c2.Log(context.Background(), log.LevelInfo, "m")

	if k := r.records[0].attrs[1].Key; k != "b" {
		t.Fatalf("c1 leaked: %q", k)
	}
	if k := r.records[1].attrs[1].Key; k != "c" {
		t.Fatalf("c2 leaked: %q", k)
	}
}

func TestWithNoAttrsReturnsSame(t *testing.T) {
	r := &capture{}
	if log.With(r) != log.Logger(r) {
		t.Fatal("With with no attrs should return the same logger")
	}
}

func TestNamedWithoutNamerIsNoop(t *testing.T) {
	r := &capture{}
	if log.Named(r, "x") != log.Logger(r) {
		t.Fatal("Named on a plain logger should return it unchanged")
	}
}

type namer struct {
	*capture
	name string
}

func (n namer) Named(name string) log.Logger { return namer{capture: n.capture, name: name} }

func TestNamedUsesNamer(t *testing.T) {
	n := namer{capture: &capture{}}
	got, ok := log.Named(n, "svc").(namer)
	if !ok || got.name != "svc" {
		t.Fatalf("Named did not use Namer: %#v", got)
	}
}

func TestGroupValue(t *testing.T) {
	a := log.Group("g", log.Int("x", 1), log.String("y", "z"))
	if a.Key != "g" || a.Value.Kind() != log.KindGroup {
		t.Fatalf("unexpected group attr: %#v", a)
	}
	if g := a.Value.Group(); len(g) != 2 || g[0].Key != "x" || g[1].Key != "y" {
		t.Fatalf("unexpected group children: %v", g)
	}
}

// skipLogger records the caller-skip carried by the logger that handled each
// Log call. It implements Wither/Namer (returning itself) so Helper.With and
// Helper.Named keep mapping onto a CallerSkipper rather than a plain wrapper.
type skipLogger struct {
	skip   int
	logged *[]int
}

func (s skipLogger) Enabled(context.Context, log.Level) bool { return true }

func (s skipLogger) Log(_ context.Context, _ log.Level, _ string, _ ...log.Attr) {
	*s.logged = append(*s.logged, s.skip)
}

func (s skipLogger) With(_ ...log.Attr) log.Logger { return s }
func (s skipLogger) Named(_ string) log.Logger     { return s }
func (s skipLogger) WithCallerSkip(skip int) log.Logger {
	return skipLogger{skip: s.skip + skip, logged: s.logged}
}

func TestAddCallerSkip(t *testing.T) {
	var logged []int
	base := skipLogger{logged: &logged}

	if got := log.AddCallerSkip(base, 0); got != log.Logger(base) {
		t.Errorf("skip 0 should return the logger unchanged")
	}

	plain := &capture{}
	if got := log.AddCallerSkip(plain, 3); got != log.Logger(plain) {
		t.Errorf("non-skipper should be returned unchanged")
	}

	log.AddCallerSkip(base, 2).Log(context.Background(), log.LevelInfo, "m")
	if len(logged) != 1 || logged[0] != 2 {
		t.Errorf("logged = %v, want [2]", logged)
	}
}

func TestHelperAddsCallerSkip(t *testing.T) {
	var logged []int
	base := skipLogger{logged: &logged}
	ctx := context.Background()

	h := log.For(base)
	h.Info(ctx, "a")                            // Helper adds one frame: skip 1
	h.With(log.String("k", "v")).Warn(ctx, "b") // skip preserved through With
	h.Named("n").Error(ctx, "c")                // skip preserved through Named
	base.Log(ctx, log.LevelInfo, "d")           // direct: skip 0

	want := []int{1, 1, 1, 0}
	if len(logged) != len(want) {
		t.Fatalf("logged = %v, want %v", logged, want)
	}
	for i := range want {
		if logged[i] != want[i] {
			t.Errorf("logged[%d] = %d, want %d", i, logged[i], want[i])
		}
	}
}
