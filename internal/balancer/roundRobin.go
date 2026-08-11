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

func (rr *roundRobin) Next() *backend.Backend {
	if len(rr.backends) == 0 {
		return nil
	}
	for i := 0; i < len(rr.backends); i++ {
		idx := rr.counter.Add(1) - 1
		ind := idx % uint64(len(rr.backends))
		if rr.backends[ind].AllowRequest() {
			return rr.backends[ind]
		}
	}
	return nil
}
