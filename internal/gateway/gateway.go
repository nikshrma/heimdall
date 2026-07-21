// Package gateway is the main dispatch package
package gateway

import (
	"net/http"
	"strings"

	"github.com/nikshrma/heimdall/internal/router"
	"github.com/rs/zerolog/log"
)

type Gateway struct {
	routes []*router.Route
}

func New(routes []*router.Route) *Gateway {
	return &Gateway{
		routes: routes,
	}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, err := router.Match(g.routes, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
	}
	if route == nil {
		http.NotFound(w, r)
	}
	b := route.Balancer.Next()
	if b == nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
	}
	if route.StripPrefix == true {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, route.Path)
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
	}
	log.Info().
		Str("url", r.URL.Path).
		Str("backend", b.URL.String()).
		Msg("dispatching request")
	b.Proxy.ServeHTTP(w, r)
}
