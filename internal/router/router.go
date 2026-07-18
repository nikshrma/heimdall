// Package router contains the runtime route structs parsed from the YAML
package router

import (
	"net/http/httputil"
	"net/url"

	"github.com/nikshrma/heimdall/internal/balancer"
	"github.com/nikshrma/heimdall/internal/config"
	"github.com/rs/zerolog/log"
)

type Backend struct {
	URL   *url.URL
	Proxy *httputil.ReverseProxy
}
type Routes struct {
	Path        string
	Methods     []string
	StripPrefix bool
	bala        balancer.Balancer
	backends    []Backend
}

func buildBackends(backends []string) []*Backend {
	var runtimeBackends []*Backend
	for _, be := range backends {
		target, err := url.Parse(be)
		if err != nil {
			log.Fatal().Err(err).Msg("invalid backend url")
		}
		proxy := &httputil.ReverseProxy{
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(target)
				pr.SetXForwarded()
			},
		}
		runtimeBackends = append(runtimeBackends, &Backend{
			URL:   target,
			Proxy: proxy,
		})
	}
	return runtimeBackends
}

func Build(cfg config.Config) ([]*Routes, error) {
	for _, rc := range cfg.Routes {
		backends := buildBackends(rc.Backends)
	}
}
