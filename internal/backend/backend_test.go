package backend

import "testing"

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

	if be.URL.Host != "localhost:8080" {
		t.Errorf("expected host localhost:8080, got %s", be.URL.Host)
	}
}
