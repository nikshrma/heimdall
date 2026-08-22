// Package gateway is the main dispatch package
package gateway

import (
	"net/http"
	"strings"

	"github.com/nikshrma/heimdall/internal/retry"
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
		return
	}
	if route == nil {
		log.Debug().Msg("gateway returned 404")
		http.NotFound(w, r)
		return
	}
	// log.Debug().
	// 	Bool("route_nil", route == nil).
	// 	Bool("balancer_nil", route != nil && route.Balancer == nil).
	// 	Msg("debug")
	if route.StripPrefix {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, route.Path)
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
	}
	retry.Retry(w, r, route)
}
