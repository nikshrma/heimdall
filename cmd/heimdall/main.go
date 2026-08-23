// Package main is the main http server for the proxy
package main

import (
	"net/http"
	"time"

	"github.com/nikshrma/heimdall/internal/config"
	"github.com/nikshrma/heimdall/internal/gateway"
	ratelimit "github.com/nikshrma/heimdall/internal/rate-limit"
	"github.com/nikshrma/heimdall/internal/router"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load config
	cfg, err := config.Load("configs/routes.yml")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read config file")
	}

	// set global log level
	level, err := zerolog.ParseLevel(cfg.Log.Level)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid log level")
	}
	zerolog.SetGlobalLevel(level)
	log.Debug().Interface("config", cfg).Msg("loaded config")

	// load dynamic routes
	routes, err := router.Build(*cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build routes")
	}
	// TODO: add config for these policy vars
	// create new limiter
	l := ratelimit.NewLimiter(32, 20, 5, time.Minute*10)

	// Use gateway
	gw := gateway.New(routes, l)
	mux := http.NewServeMux()
	mux.Handle("/", gw)

	log.Info().Msg("starting server")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
