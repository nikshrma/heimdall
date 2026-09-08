// Package gateway is the main dispatch package
package gateway

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nikshrma/heimdall/internal/ctxkeys"
	"github.com/nikshrma/heimdall/internal/metrics"
	ratelimit "github.com/nikshrma/heimdall/internal/rate-limit"
	"github.com/nikshrma/heimdall/internal/router"
	"github.com/rs/zerolog/log"
)

type Gateway struct {
	routes []*router.Route
	l      *ratelimit.Limiter
}
type StatusWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (w *StatusWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}

	w.statusCode = code
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *StatusWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(p)
}

func (w *StatusWriter) StatusCode() int {
	if !w.wroteHeader {
		return http.StatusOK
	}
	return w.statusCode
}

func New(routes []*router.Route, l *ratelimit.Limiter) *Gateway {
	return &Gateway{
		routes: routes,
		l:      l,
	}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, err := router.Match(g.routes, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
		log.Warn().Str("url", r.URL.Path).Msg("request failed: method not allowed")
		return
	}
	if route == nil {
		log.Debug().Msg("gateway returned 404")
		http.NotFound(w, r)
		log.Warn().Str("url", r.URL.Path).Msg("request failed: no matching route")
		return
	}
	if route.StripPrefix {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, route.Path)
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
	}
	ctx := context.WithValue(r.Context(), ctxkeys.RouteKey{}, route)
	ctx = context.WithValue(ctx, ctxkeys.RoutePathKey{}, route.Path)
	r = r.WithContext(ctx)

	sw := &StatusWriter{
		ResponseWriter: w,
	}

	metrics.InFlightRequests.WithLabelValues(route.Path).Inc()
	defer metrics.InFlightRequests.WithLabelValues(route.Path).Dec()

	start := time.Now()

	g.l.RateLimit(sw, r)
	duration := time.Since(start).Seconds()
	metrics.TotalRequests.WithLabelValues(route.Path, r.Method, strconv.Itoa(sw.StatusCode())).Inc()
	metrics.RequestDuration.WithLabelValues(route.Path, r.Method).Observe(duration)
}
