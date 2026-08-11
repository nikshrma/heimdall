package backend

import (
	"net/http"
	"time"
)

type breakerTransport struct {
	next http.RoundTripper
	b    *Backend
}

func (t *breakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.next.RoundTrip(req)
	duration := time.Since(start)
	if err != nil || resp != nil && resp.StatusCode >= 500 || duration > t.b.slowThreshold {
		t.b.MarkFailure()
	} else {
		t.b.MarkSuccess()
	}
	if t.b.state.Load() == int32(HalfOpen) {
		t.b.trialInFlight.Store(false)
	}
	return resp, err
}
