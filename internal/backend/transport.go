package backend

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type breakerTransport struct {
	next http.RoundTripper
	b    *Backend
}

func (t *breakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	resp, err := t.next.RoundTrip(req)
	duration := time.Since(start)

	switch {
	case err != nil:
		log.Info().
			Str("backend", t.b.URL().String()).
			Err(err).
			Dur("duration", duration).
			Msg("breaker: fail (transport error)")

		t.b.MarkFailure()

	case resp != nil && resp.StatusCode >= 500:
		log.Info().
			Str("backend", t.b.URL().String()).
			Int("status", resp.StatusCode).
			Dur("duration", duration).
			Msg("breaker: fail (5xx)")

		t.b.MarkFailure()

	case duration > t.b.slowThreshold:
		log.Info().
			Str("backend", t.b.URL().String()).
			Dur("duration", duration).
			Msg("breaker: fail (slow)")

		t.b.MarkFailure()

	default:
		t.b.MarkSuccess()
	}

	if t.b.state.Load() == int32(HalfOpen) {
		t.b.trialInFlight.Store(false)
	}

	return resp, err
}
