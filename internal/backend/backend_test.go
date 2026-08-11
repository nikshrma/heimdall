package backend

import (
	"sync"
	"testing"
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
		t.Error("expected new backend to start Closed")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := New("://not-a-valid-url")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestMarkFailure_BecomesOpenAfterThreeFailures(t *testing.T) {
	be, _ := New("http://localhost:8080")

	if State(be.state.Load()) != Closed {
		t.Fatal("expected backend to start Closed")
	}

	be.MarkFailure()
	be.MarkFailure()

	if State(be.state.Load()) != Closed {
		t.Error("expected backend to still be Closed after 2 failures")
	}

	be.MarkFailure()

	if State(be.state.Load()) != Open {
		t.Error("expected backend to be Open after 3 failures")
	}
}

func TestAllowRequest_Closed(t *testing.T) {
	be, _ := New("http://localhost:8080")

	if !be.AllowRequest() {
		t.Error("expected Closed backend to allow request")
	}
}

func TestAllowRequest_Open(t *testing.T) {
	be, _ := New("http://localhost:8080")

	be.state.Store(int32(Open))

	if be.AllowRequest() {
		t.Error("expected Open backend to reject request")
	}
}

func TestAllowRequest_HalfOpen_AllowsOnlyOne(t *testing.T) {
	be, _ := New("http://localhost:8080")

	be.state.Store(int32(HalfOpen))

	if !be.AllowRequest() {
		t.Fatal("expected first HalfOpen request to be allowed")
	}

	if be.AllowRequest() {
		t.Error("expected second HalfOpen request to be rejected")
	}

	if be.AllowRequest() {
		t.Error("expected third HalfOpen request to be rejected")
	}
}

func TestMarkSuccess_HalfOpenBecomesClosedAfterThree(t *testing.T) {
	be, _ := New("http://localhost:8080")

	be.state.Store(int32(HalfOpen))

	be.MarkSuccess()

	if State(be.state.Load()) != HalfOpen {
		t.Error("expected backend to remain HalfOpen after 1 success")
	}

	be.MarkSuccess()

	if State(be.state.Load()) != HalfOpen {
		t.Error("expected backend to remain HalfOpen after 2 successes")
	}

	be.MarkSuccess()

	if State(be.state.Load()) != Closed {
		t.Error("expected backend to become Closed after 3 successes")
	}

	if be.successCount.Load() != 0 {
		t.Error("expected success count to reset after becoming Closed")
	}

	if be.failureCount.Load() != 0 {
		t.Error("expected failure count to reset after becoming Closed")
	}
}

func TestMarkFailure_HalfOpenBecomesOpen(t *testing.T) {
	be, _ := New("http://localhost:8080")

	be.state.Store(int32(HalfOpen))

	be.MarkFailure()

	if State(be.state.Load()) != Open {
		t.Error("expected HalfOpen backend to become Open after failure")
	}

	if be.successCount.Load() != 0 {
		t.Error("expected success count to reset after failure")
	}
}

func TestMarkSuccess_ClosedResetsFailures(t *testing.T) {
	be, _ := New("http://localhost:8080")

	be.MarkFailure()
	be.MarkFailure()

	if be.failureCount.Load() != 2 {
		t.Fatalf("expected 2 failures, got %d", be.failureCount.Load())
	}

	be.MarkSuccess()

	if be.failureCount.Load() != 0 {
		t.Errorf(
			"expected failure count to reset after success, got %d",
			be.failureCount.Load(),
		)
	}

	if State(be.state.Load()) != Closed {
		t.Error("expected backend to remain Closed")
	}
}

func TestHalfOpenTrialCanBeReleased(t *testing.T) {
	be, _ := New("http://localhost:8080")

	be.state.Store(int32(HalfOpen))

	if !be.AllowRequest() {
		t.Fatal("expected first trial request to be allowed")
	}

	if be.AllowRequest() {
		t.Fatal("expected second request to be rejected while trial is active")
	}

	be.trialInFlight.Store(false)

	if !be.AllowRequest() {
		t.Error("expected another trial after previous trial was released")
	}
}

func TestBackendConcurrent(t *testing.T) {
	be, _ := New("http://localhost:8080")

	var wg sync.WaitGroup

	const goroutines = 200
	const iterations = 10000

	for g := 0; g < goroutines; g++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for i := 0; i < iterations; i++ {
				switch (i + id) % 3 {
				case 0:
					be.MarkFailure()
				default:
					be.MarkSuccess()
				}

				_ = be.AllowRequest()
				_ = be.URL()
			}
		}(g)
	}

	wg.Wait()
}
