// Package retry is the layer responsible for request retries
package retry

import (
	"net/http"
	"strconv"

	"github.com/nikshrma/heimdall/internal/backend"
	"github.com/nikshrma/heimdall/internal/ctxkeys"
	"github.com/nikshrma/heimdall/internal/metrics"
	"github.com/nikshrma/heimdall/internal/router"
	"github.com/rs/zerolog/log"
)

func Retry(w http.ResponseWriter, r *http.Request) {
	route := r.Context().Value(ctxkeys.RouteKey{}).(*router.Route)
	if !ShouldRetryMethod(r.Method) {
		b := route.Balancer.Next(nil)
		if b == nil {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		metrics.ProxyBackendSelectedTotal.WithLabelValues(b.URL().String(), route.Path).Inc()
		log.Info().
			Str("url", r.URL.Path).Str("backend", b.URL().String()).Msg("proxied non-retryable request")
		b.Proxy.ServeHTTP(w, r)
		return
	}
	p := NewPolicy()
	excluded := make(map[*backend.Backend]struct{})
	var lastBuffer *ResponseBuffer
	retryCount := 0
	for p.AttemptAgain() {
		b := route.Balancer.Next(excluded)
		if b == nil {
			if lastBuffer != nil {
				log.Warn().
					Str("url", r.URL.Path).Msg("request failed: no backends available for retry")
				setRetryCountHeader(lastBuffer, retryCount)
				lastBuffer.WriteTo(w)
				return
			} else {
				log.Warn().
					Str("url", r.URL.Path).Msg("request failed: bad gateway")
				http.Error(w, "bad gateway", http.StatusBadGateway)
				return
			}
		}
		metrics.ProxyBackendSelectedTotal.WithLabelValues(b.URL().String(), route.Path).Inc()
		buffer := NewResponseBuffer()
		b.Proxy.ServeHTTP(buffer, r)
		lastBuffer = buffer
		if !ShouldRetryStatus(buffer.StatusCode()) {
			log.Info().
				Str("url", r.URL.Path).Str("backend", b.URL().String()).Msg("request complete")
			setRetryCountHeader(buffer, retryCount)
			buffer.WriteTo(w)
			return
		}
		retryCount++
		metrics.ProxyRetriesTotal.WithLabelValues(b.URL().String(), route.Path).Inc()
		excluded[b] = struct{}{}
	}
	if lastBuffer != nil {
		log.Warn().
			Str("url", r.URL.Path).Msg("request failed: ran out of retry attempts")
		setRetryCountHeader(lastBuffer, retryCount)
		lastBuffer.WriteTo(w)
		metrics.ProxyRequestsExhaustedRetriesTotal.WithLabelValues(route.Path).Inc()
		return
	} else {
		log.Warn().
			Str("url", r.URL.Path).Msg("request failed: bad gateway")
		http.Error(w, "bad gateway", http.StatusBadGateway)
		metrics.ProxyRequestsExhaustedRetriesTotal.WithLabelValues(route.Path).Inc()
	}
}

func setRetryCountHeader(buffer *ResponseBuffer, retryCount int) {
	if retryCount > 0 {
		buffer.Header().Set("X-Retry-Count", strconv.Itoa(retryCount))
	}
}
