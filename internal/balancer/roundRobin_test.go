package balancer

import (
	"testing"

	"github.com/nikshrma/heimdall/internal/backend"
)

func TestRoundRobinNext(t *testing.T) {
	backends := []*backend.Backend{
		{},
		{},
		{},
	}
	rr := NewRoundRobin(backends)
	if rr.Next() != backends[0] {
		t.Error("expected first backend")
	}
}
