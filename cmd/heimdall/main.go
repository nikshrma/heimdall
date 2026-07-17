// Package main is the main http server for the proxy
package main

import (
	"net/http"

	"github.com/nikshrma/heimdall/internal/config"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.Load("configs/routes.yml")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read config file")
	}
	log.Info().Interface("config", cfg).Msg("loaded config")
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	})
	log.Info().Msg("starting server")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
