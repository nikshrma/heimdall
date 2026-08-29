// Package gateway is the main dispatch package
package gateway

import (
	"net/http"
	"strings"

	ratelimit "github.com/nikshrma/heimdall/internal/rate-limit"
	"github.com/nikshrma/heimdall/internal/router"
	"github.com/rs/zerolog/log"
)

type Gateway struct {
	routes []*router.Route
	l      *ratelimit.Limiter
}

func New(routes []*router.Route, l *ratelimit.Limiter) *Gateway {
	return &Gateway{
		routes: routes,
		l:      l,
	}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: decide defaults for limiter if not initialised in main.go
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
	g.l.RateLimit(w, r, route)
}
