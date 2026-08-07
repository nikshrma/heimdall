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

	if rr.Next() != backends[0] {
		t.Fatal("expected first backend")
	}

	if rr.Next() != backends[1] {
		t.Fatal("expected second backend")
	}

	if rr.Next() != backends[2] {
		t.Fatal("expected third backend")
	}

	if rr.Next() != backends[0] {
		t.Fatal("expected first backend again")
	}
}

func TestRoundRobinEmpty(t *testing.T) {
	rr := NewRoundRobin(nil)

	if rr.Next() != nil {
		t.Fatal("expected nil backend")
	}
}

func TestRoundRobinSkipsUnhealthy(t *testing.T) {
	b1 := mustBackend(t, "http://backend1")
	b2 := mustBackend(t, "http://backend2")
	b3 := mustBackend(t, "http://backend3")

	// mark backend2 unhealthy
	b2.MarkFailure()
	b2.MarkFailure()
	b2.MarkFailure()

	rr := NewRoundRobin([]*backend.Backend{
		b1,
		b2,
		b3,
	})

	hits := make(map[*backend.Backend]int)
	for i := 0; i < 10; i++ {
		if b := rr.Next(); b != nil {
			hits[b]++
		}
	}

	if hits[b2] != 0 {
		t.Errorf("expected 0 hits for unhealthy backend2, got %d", hits[b2])
	}
	if hits[b1] == 0 {
		t.Error("expected hits for backend1, got 0")
	}
	if hits[b3] == 0 {
		t.Error("expected hits for backend3, got 0")
	}
}

func TestRoundRobinAllUnhealthy(t *testing.T) {
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

	if rr.Next() != nil {
		t.Fatal("expected nil when every backend is unhealthy")
	}
}
