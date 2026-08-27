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
	n := len(rr.backends)
	if n == 0 {
		return nil
	}
	checked := make([]uint64, (n+63)/64)

	for checkedCount := 0; checkedCount < n; {
		val := rr.counter.Add(1) - 1
		ind := int(val % uint64(n))

		word := ind / 64
		bit := ind % 64

		if checked[word]&(uint64(1)<<bit) != 0 {
			continue
		}

		checked[word] |= uint64(1) << bit
		checkedCount++

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
