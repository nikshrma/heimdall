package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nikshrma/heimdall/internal/router"
)

func TestLimiterAllow(t *testing.T) {
	l := NewLimiter(4, 2, 0, time.Minute)

	if !l.Allow("127.0.0.1") {
		t.Fatal("first request should be allowed")
	}

	if !l.Allow("127.0.0.1") {
		t.Fatal("second request should be allowed")
	}

	if l.Allow("127.0.0.1") {
		t.Fatal("third request should be rejected")
	}
}

func TestLimiterRefill(t *testing.T) {
	l := NewLimiter(4, 1, 10, time.Minute)

	if !l.Allow("127.0.0.1") {
		t.Fatal("first request should be allowed")
	}

	if l.Allow("127.0.0.1") {
		t.Fatal("second request should be rejected")
	}

	time.Sleep(150 * time.Millisecond)

	if !l.Allow("127.0.0.1") {
		t.Fatal("token should have refilled")
	}
}

func TestLimiterDifferentIPs(t *testing.T) {
	l := NewLimiter(4, 1, 0, time.Minute)

	if !l.Allow("127.0.0.1") {
		t.Fatal("first IP should be allowed")
	}

	if !l.Allow("127.0.0.2") {
		t.Fatal("different IP should have its own bucket")
	}
}

func TestLimiterRateLimitInvalidAddress(t *testing.T) {
	l := NewLimiter(4, 1, 0, time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "invalid-address"

	rec := httptest.NewRecorder()

	l.RateLimit(rec, req, nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestLimiterRateLimitExceeded(t *testing.T) {
	l := NewLimiter(4, 1, 0, time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	// We don't want Retry to matter here; exhaust the bucket first.
	if !l.Allow("127.0.0.1") {
		t.Fatal("initial request should be allowed")
	}

	rec := httptest.NewRecorder()
	l.RateLimit(rec, req, (*router.Route)(nil))

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected %d, got %d", http.StatusTooManyRequests, rec.Code)
	}
}

func TestLimiterCleanup(t *testing.T) {
	l := NewLimiter(2, 1, 0, 50*time.Millisecond)

	addr := "127.0.0.1"
	l.Allow(addr)

	found := false
	for i := range l.shards {
		l.shards[i].mu.RLock()
		_, ok := l.shards[i].buckets[addr]
		l.shards[i].mu.RUnlock()

		if ok {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("expected bucket to exist")
	}

	// Cleanup runs every ttl/2 and removes buckets older than ttl.
	time.Sleep(150 * time.Millisecond)

	found = false
	for i := range l.shards {
		l.shards[i].mu.RLock()
		_, ok := l.shards[i].buckets[addr]
		l.shards[i].mu.RUnlock()

		if ok {
			found = true
			break
		}
	}

	if found {
		t.Fatal("expected expired bucket to be cleaned up")
	}
}
