package backend

import (
	"sync"
	"testing"
	"time"
)

func TestNew_ValidURL(t *testing.T) {
	be, err := New("http://localhost:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if be == nil {
		t.Fatal("expected backend, got nil")
	}

	if be.Proxy == nil {
		t.Error("expected proxy to be initialized")
	}

	if be.URL().Host != "localhost:8080" {
		t.Errorf("expected host localhost:8080, got %s", be.URL().Host)
	}

	if State(be.state.Load()) != Closed {
		t.Error("expected backend to start Closed")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := New("://not-a-valid-url")

	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestClosedToOpenAfterFailures(t *testing.T) {
	be, _ := New("http://localhost:8080")

	be.MarkFailure()
	be.MarkFailure()

	if State(be.state.Load()) != Closed {
		t.Fatal("expected backend to remain Closed after 2 failures")
	}

	be.MarkFailure()

	if State(be.state.Load()) != Open {
		t.Error("expected backend to become Open after 3 failures")
	}
}

func TestSuccessResetsFailures(t *testing.T) {
	be, _ := New("http://localhost:8080")

	be.MarkFailure()
	be.MarkFailure()

	be.MarkSuccess()

	if be.failureCount.Load() != 0 {
		t.Errorf("expected failures to reset, got %d", be.failureCount.Load())
	}

	if State(be.state.Load()) != Closed {
		t.Error("expected backend to remain Closed")
	}
}

func TestAllowRequest(t *testing.T) {
	be, _ := New("http://localhost:8080")

	// Closed allows requests.
	if !be.AllowRequest() {
		t.Error("expected Closed backend to allow requests")
	}

	// Open rejects requests.
	be.state.Store(int32(Open))

	if be.AllowRequest() {
		t.Error("expected Open backend to reject requests")
	}

	// HalfOpen allows only one trial.
	be.state.Store(int32(HalfOpen))
	be.trialInFlight.Store(false)

	if !be.AllowRequest() {
		t.Fatal("expected first HalfOpen request to be allowed")
	}

	if be.AllowRequest() {
		t.Error("expected second HalfOpen request to be rejected")
	}
}

func TestHalfOpenToClosedAfterSuccesses(t *testing.T) {
	be, _ := New("http://localhost:8080")

	be.state.Store(int32(HalfOpen))

	be.MarkSuccess()
	be.MarkSuccess()

	if State(be.state.Load()) != HalfOpen {
		t.Fatal("expected backend to remain HalfOpen after 2 successes")
	}

	be.MarkSuccess()

	if State(be.state.Load()) != Closed {
		t.Error("expected backend to become Closed after 3 successes")
	}

	if be.successCount.Load() != 0 {
		t.Error("expected success count to reset")
	}

	if be.failureCount.Load() != 0 {
		t.Error("expected failure count to reset")
	}
}

func TestHalfOpenFailureTripsBackend(t *testing.T) {
	be, _ := New("http://localhost:8080")

	be.state.Store(int32(HalfOpen))
	be.MarkFailure()

	if State(be.state.Load()) != Open {
		t.Error("expected HalfOpen backend to become Open after failure")
	}

	if be.successCount.Load() != 0 {
		t.Error("expected success count to reset")
	}
}

func TestOpenBecomesHalfOpenAfterCooldown(t *testing.T) {
	be, _ := New("http://localhost:8080")
	be.cooldown = 10 * time.Millisecond

	be.trip()

	if State(be.state.Load()) != Open {
		t.Fatal("expected backend to be Open immediately")
	}

	time.Sleep(20 * time.Millisecond)

	if State(be.state.Load()) != HalfOpen {
		t.Error("expected backend to become HalfOpen after cooldown")
	}
}

func TestBackendConcurrent(t *testing.T) {
	be, _ := New("http://localhost:8080")

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for j := 0; j < 1000; j++ {
				if (id+j)%2 == 0 {
					be.MarkSuccess()
				} else {
					be.MarkFailure()
				}

				be.AllowRequest()
				be.URL()
			}
		}(i)
	}

	wg.Wait()
}
