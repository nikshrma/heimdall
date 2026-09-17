package backend

import (
	"net/http"
	"strconv"
	"time"

	"github.com/nikshrma/heimdall/internal/ctxkeys"
	"github.com/nikshrma/heimdall/internal/metrics"
	"github.com/rs/zerolog/log"
)

type breakerTransport struct {
	next http.RoundTripper
	b    *Backend
}

func (t *breakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	path := req.Context().Value(ctxkeys.RoutePathKey{}).(string)

	start := time.Now()

	metrics.ProxyBackendInFlightRequests.WithLabelValues(t.b.URL().String()).Inc()

	resp, err := t.next.RoundTrip(req)
	duration := time.Since(start)

	statusCode := "0"
	if resp != nil {
		statusCode = strconv.Itoa(resp.StatusCode)
	}

	metrics.ProxyBackendRequestsTotal.WithLabelValues(t.b.URL().String(), path, req.Method, statusCode).Inc()
	metrics.ProxyBackendInFlightRequests.WithLabelValues(t.b.URL().String()).Dec()
	metrics.ProxyBackendRequestDuration.WithLabelValues(t.b.URL().String(), path).Observe(duration.Seconds())

	if !t.b.enabled {
		return resp, err
	}

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
