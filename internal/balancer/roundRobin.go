package balancer

import (
	"sync/atomic"

	"github.com/nikshrma/heimdall/internal/backend"
)

type roundRobin struct {
	backends []*backend.Backend
	counter  atomic.Uint64
}

func NewRoundRobin(backends []*backend.Backend) Balancer {
	return &roundRobin{
		backends: backends,
	}
}

func (rr *roundRobin) Next(excluded map[*backend.Backend]struct{}) *backend.Backend {
	if len(rr.backends) == 0 {
		return nil
	}
	n := uint64(len(rr.backends))
	start := rr.counter.Add(1) - 1
	for i := uint64(0); i < n; i++ {
		ind := (start + i) % n
		b := rr.backends[ind]
		if _, ok := excluded[b]; ok {
			continue
		}
		if !b.AllowRequest() {
			continue
		}
		return b
	}
	return nil
}
