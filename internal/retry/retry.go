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
		log.Debug().
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
				lastBuffer.WriteTo(w)
				return
			} else {
				http.Error(w, "bad gateway", http.StatusBadGateway)
				return
			}
		}
		buffer := NewResponseBuffer()
		b.Proxy.ServeHTTP(buffer, r)
		lastBuffer = buffer
		if !ShouldRetryStatus(buffer.StatusCode()) {
			buffer.WriteTo(w)
			return
		}
		excluded[b] = struct{}{}
	}
	if lastBuffer != nil {
		lastBuffer.WriteTo(w)
		return
	} else {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}
}
