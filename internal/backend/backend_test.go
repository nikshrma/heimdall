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

	if !be.IsHealthy() {
		t.Error("expected new backend to start healthy")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := New("://not-a-valid-url")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestMarkSuccess_BecomesHealthyAfterThree(t *testing.T) {
	be, _ := New("http://localhost:8080")

	be.MarkFailure()
	be.MarkFailure()
	be.MarkFailure()

	if be.IsHealthy() {
		t.Fatal("expected backend to be unhealthy")
	}

	be.MarkSuccess()
	be.MarkSuccess()

	if be.IsHealthy() {
		t.Error("expected backend to still be unhealthy after 2 successes")
	}

	be.MarkSuccess()

	if !be.IsHealthy() {
		t.Error("expected backend to be healthy after 3 successes")
	}
}

func TestMarkFailure_BecomesUnhealthyAfterThree(t *testing.T) {
	be, _ := New("http://localhost:8080")

	if !be.IsHealthy() {
		t.Fatal("expected backend to start healthy")
	}

	be.MarkFailure()
	be.MarkFailure()

	if !be.IsHealthy() {
		t.Error("expected backend to still be healthy after 2 failures")
	}

	be.MarkFailure()

	if be.IsHealthy() {
		t.Error("expected backend to be unhealthy after 3 failures")
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

				_ = be.IsHealthy()
				_ = be.URL()
			}
		}(g)
	}

	wg.Wait()
}
