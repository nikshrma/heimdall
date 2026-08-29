// Package retry is the layer responsible for request retries
package retry

import (
	"net/http"

	"github.com/nikshrma/heimdall/internal/backend"
	"github.com/nikshrma/heimdall/internal/router"
	"github.com/rs/zerolog/log"
)

func Retry(w http.ResponseWriter, r *http.Request, route *router.Route) {
	if !ShouldRetryMethod(r.Method) {
		b := route.Balancer.Next(nil)
		if b == nil {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		log.Info().
			Str("url", r.URL.Path).Str("backend", b.URL().String()).Msg("proxied non-retryable request")
		b.Proxy.ServeHTTP(w, r)
		return
	}
	p := NewPolicy()
	excluded := make(map[*backend.Backend]struct{})
	var lastBuffer *ResponseBuffer
	for p.AttemptAgain() {
		b := route.Balancer.Next(excluded)
		if b == nil {
			if lastBuffer != nil {
				log.Warn().
					Str("url", r.URL.Path).Msg("request failed: no backends available for retry")
				lastBuffer.WriteTo(w)
				return
			} else {
				log.Warn().
					Str("url", r.URL.Path).Msg("request failed: bad gateway")
				http.Error(w, "bad gateway", http.StatusBadGateway)
				return
			}
		}
		buffer := NewResponseBuffer()
		b.Proxy.ServeHTTP(buffer, r)
		lastBuffer = buffer
		if !ShouldRetryStatus(buffer.StatusCode()) {
			log.Info().
				Str("url", r.URL.Path).Str("backend", b.URL().String()).Msg("request complete")
			buffer.WriteTo(w)
			return
		}
		excluded[b] = struct{}{}
	}
	if lastBuffer != nil {
		log.Warn().
			Str("url", r.URL.Path).Msg("request failed: ran out of retry attempts")
		lastBuffer.WriteTo(w)
		return
	} else {
		log.Warn().
			Str("url", r.URL.Path).Msg("request failed: bad gateway")
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}
}
