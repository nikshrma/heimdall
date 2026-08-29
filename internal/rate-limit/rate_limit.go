// Package ratelimit is responsible for IP based rate-limiting
package ratelimit

import (
	"hash/fnv"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nikshrma/heimdall/internal/retry"
	"github.com/nikshrma/heimdall/internal/router"
	"github.com/rs/zerolog/log"
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
	lastUsed atomic.Int64
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

	s.mu.RLock()
	if b, ok := s.buckets[addr]; ok {
		s.mu.RUnlock()
		return b
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	b := &bucket{
		tokens: l.capacity,
	}
	b.lastUsed.Store(time.Now().UnixNano())
	s.buckets[addr] = b

	return b
}

func (l *Limiter) Allow(addr string) bool {
	b := l.getOrCreateBucket(addr)
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	timePassed := b.idleFor()
	b.tokens = min(l.capacity, timePassed.Seconds()*l.refillRate+b.tokens)
	b.lastUsed.Store(now.UnixNano())

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
		log.Info().Err(err).Msg("invalid client address")
		return
	}
	if !l.Allow(addr) {
		http.Error(w, "You've been rate limitted", http.StatusTooManyRequests)
		log.Warn().Str("url", r.URL.Path).Msg("request has been rate-limited")
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
			s.mu.RLock()
			stale := make([]string, 0, len(s.buckets)/4)
			for addr, b := range s.buckets {
				if b.idleFor() > l.ttl/4 {
					stale = append(stale, addr)
				}
			}
			s.mu.RUnlock()
			if len(stale) == 0 {
				continue
			}
			s.mu.Lock()
			for _, addr := range stale {
				if b, ok := s.buckets[addr]; ok && b.idleFor() > l.ttl/4 {
					delete(s.buckets, addr)
				}
			}
			s.mu.Unlock()
		}
	}
}

func (b *bucket) idleFor() time.Duration {
	return time.Since(time.Unix(0, b.lastUsed.Load()))
}
