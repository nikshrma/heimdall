// Package router contains the runtime route structs parsed from the YAML
package router

import (
	"github.com/nikshrma/heimdall/internal/backend"
	"github.com/nikshrma/heimdall/internal/balancer"
	"github.com/nikshrma/heimdall/internal/config"
)

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
	return runtimeRoutes, nil
}
