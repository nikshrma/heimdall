package balancer

import (
	"testing"

	"github.com/nikshrma/heimdall/internal/backend"
)

func mustBackend(t *testing.T, raw string) *backend.Backend {
	t.Helper()

	b, err := backend.New(raw)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	return b
}

func TestRoundRobinNext(t *testing.T) {
	backends := []*backend.Backend{
		mustBackend(t, "http://backend1"),
		mustBackend(t, "http://backend2"),
		mustBackend(t, "http://backend3"),
	}

	rr := NewRoundRobin(backends)

	if rr.Next(nil) != backends[0] {
		t.Fatal("expected first backend")
	}

	if rr.Next(nil) != backends[1] {
		t.Fatal("expected second backend")
	}

	if rr.Next(nil) != backends[2] {
		t.Fatal("expected third backend")
	}

	if rr.Next(nil) != backends[0] {
		t.Fatal("expected first backend again")
	}
}

func TestRoundRobinEmpty(t *testing.T) {
	rr := NewRoundRobin(nil)

	if rr.Next(nil) != nil {
		t.Fatal("expected nil backend")
	}
}

func TestRoundRobinSkipsExcluded(t *testing.T) {
	b1 := mustBackend(t, "http://backend1")
	b2 := mustBackend(t, "http://backend2")
	b3 := mustBackend(t, "http://backend3")

	rr := NewRoundRobin([]*backend.Backend{
		b1,
		b2,
		b3,
	})

	excluded := map[*backend.Backend]struct{}{
		b2: {},
	}

	for i := 0; i < 10; i++ {
		b := rr.Next(excluded)

		if b == b2 {
			t.Fatal("expected excluded backend to be skipped")
		}

		if b == nil {
			t.Fatal("expected an available backend")
		}
	}
}

func TestRoundRobinSkipsUnavailable(t *testing.T) {
	b1 := mustBackend(t, "http://backend1")
	b2 := mustBackend(t, "http://backend2")

	b2.MarkFailure()
	b2.MarkFailure()
	b2.MarkFailure()

	rr := NewRoundRobin([]*backend.Backend{
		b1,
		b2,
	})

	for i := 0; i < 5; i++ {
		if b := rr.Next(nil); b != b1 {
			t.Fatal("expected unavailable backend to be skipped")
		}
	}
}

func TestRoundRobinAllExcluded(t *testing.T) {
	b1 := mustBackend(t, "http://backend1")
	b2 := mustBackend(t, "http://backend2")

	rr := NewRoundRobin([]*backend.Backend{
		b1,
		b2,
	})

	excluded := map[*backend.Backend]struct{}{
		b1: {},
		b2: {},
	}

	if rr.Next(excluded) != nil {
		t.Fatal("expected nil when every backend is excluded")
	}
}

func TestRoundRobinAllUnavailable(t *testing.T) {
	b1 := mustBackend(t, "http://backend1")
	b2 := mustBackend(t, "http://backend2")

	for _, b := range []*backend.Backend{b1, b2} {
		b.MarkFailure()
		b.MarkFailure()
		b.MarkFailure()
	}

	rr := NewRoundRobin([]*backend.Backend{
		b1,
		b2,
	})

	if rr.Next(nil) != nil {
		t.Fatal("expected nil when every backend is unavailable")
	}
}
