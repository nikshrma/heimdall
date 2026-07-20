// Package router contains the runtime route structs parsed from the YAML
package router

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/nikshrma/heimdall/internal/backend"
	"github.com/nikshrma/heimdall/internal/balancer"
	"github.com/nikshrma/heimdall/internal/config"
)

var ErrorMethodNotAllowed = errors.New("method not allowed")

type Route struct {
	Path        string
	Methods     []string
	StripPrefix bool
	Balancer    balancer.Balancer
	Backends    []*backend.Backend
}

func buildBackends(backends []string) ([]*backend.Backend, error) {
	var runtimeBackends []*backend.Backend
	for _, be := range backends {
		b, err := backend.New(be)
		if err != nil {
			return nil, err
		}
		runtimeBackends = append(runtimeBackends, b)
	}
	return runtimeBackends, nil
}

func Build(cfg config.Config) ([]*Route, error) {
	var runtimeRoutes []*Route
	for _, rc := range cfg.Routes {
		backends, err := buildBackends(rc.Backends)
		if err != nil {
			return nil, err
		}

		runtimeRoutes = append(runtimeRoutes, &Route{
			Path:        rc.Path,
			Methods:     rc.Methods,
			StripPrefix: rc.StripPrefix,
			Backends:    backends,
			Balancer:    balancer.NewRoundRobin(backends),
		})
	}
	if len(runtimeRoutes) == 0 {
		return nil, errors.New("no routes configured")
	}
	return runtimeRoutes, nil
}

func Match(routes []*Route, req *http.Request) (*Route, error) {
	var best *Route
	methodMismatch := false

	for _, rc := range routes {
		if strings.HasPrefix(req.URL.Path, rc.Path) {
			if best == nil || len(rc.Path) > len(best.Path) {
				if MethodMatch(rc.Methods, req.Method) {
					best = rc
					methodMismatch = false
				} else if best == nil {
					methodMismatch = true
				}
			}
		}
	}

	if best != nil {
		return best, nil
	}
	if methodMismatch {
		return nil, ErrorMethodNotAllowed
	}
	return nil, nil
}

func MethodMatch(methods []string, method string) bool {
	if len(methods) == 0 {
		return true // no restriction configured
	}
	return slices.Contains(methods, method)
}
