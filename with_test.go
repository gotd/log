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
