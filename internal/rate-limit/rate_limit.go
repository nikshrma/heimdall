// Package ratelimit is responsible for IP based rate-limiting
package ratelimit

import (
	"hash/fnv"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/nikshrma/heimdall/internal/retry"
	"github.com/nikshrma/heimdall/internal/router"
)

type Limiter struct {
	shards     []shard
	capacity   float64
	refillRate float64
	ttl        time.Duration
}

type bucket struct {
	mu       sync.Mutex
	tokens   float64
	lastUsed time.Time
}
type shard struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
}

func NewLimiter(numShards int, cap float64, refillRate int64, ttl time.Duration) *Limiter {
	shards := make([]shard, numShards)
	for i := range numShards {
		shards[i].buckets = make(map[string]*bucket)
	}
	l := &Limiter{
		shards:     shards,
		capacity:   cap,
		refillRate: float64(refillRate),
		ttl:        ttl,
	}
	go l.CleanUp()
	return l
}

func (l *Limiter) getOrCreateBucket(addr string) *bucket {
	h := fnv.New32a()
	h.Write([]byte(addr))
	ind := int(h.Sum32() % uint32(len(l.shards)))
	s := &l.shards[ind]

	s.mu.Lock()
	defer s.mu.Unlock()

	if b, ok := s.buckets[addr]; ok {
		return b
	}
	b := &bucket{
		tokens:   l.capacity,
		lastUsed: time.Now(),
	}
	s.buckets[addr] = b

	return b
}

func (l *Limiter) Allow(addr string) bool {
	now := time.Now()

	b := l.getOrCreateBucket(addr)
	b.mu.Lock()
	defer b.mu.Unlock()

	timePassed := time.Since(b.lastUsed).Seconds()

	b.tokens = min(l.capacity, float64(timePassed)*l.refillRate+b.tokens)
	b.lastUsed = now

	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1

	return true
}

func (l *Limiter) RateLimit(w http.ResponseWriter, r *http.Request, route *router.Route) {
	addr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, "invalid client address", http.StatusBadRequest)
		return
	}
	if !l.Allow(addr) {
		http.Error(w, "You've been rate limitted", http.StatusTooManyRequests)
		return
	}

	retry.Retry(w, r, route)
}

func (l *Limiter) CleanUp() {
	// TODO: also change a bunch of stuff here to be configurable including the ticker time for this cleanup
	tc := time.NewTicker(l.ttl / 2)
	defer tc.Stop()
	for range tc.C {
		for i := range l.shards {
			s := &l.shards[i]
			s.mu.Lock()
			for addr, b := range s.buckets {
				b.mu.Lock()
				passedTime := time.Since(b.lastUsed).Seconds()
				b.mu.Unlock()
				if passedTime > float64(l.ttl) {
					delete(s.buckets, addr)
				}
			}
		}
	}
}
