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
	if rr.Next() != backends[1] {
		t.Error("expected second backend")
	}
	if rr.Next() != backends[2] {
		t.Error("expected third backend")
	}
	if rr.Next() != backends[0] {
		t.Error("expected first backend again")
	}
}

func TestRoundRobinEmpty(t *testing.T) {
	rr := NewRoundRobin(nil)
	if rr.Next() != nil {
		t.Error("expected nil backend")
	}
}
