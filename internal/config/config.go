// Package config parses the YAML config
package config

import (
	"os"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type RouteConfig struct {
	Path        string   `yaml:"path"`
	Methods     []string `yaml:"methods"`
	StripPrefix bool     `yaml:"stripPrefix"`
	Backends    []string `yaml:"backends"`
}
type LogConfig struct {
	Level string `yaml:"level"`
}

type Config struct {
	Log    LogConfig     `yaml:"log"`
	Routes []RouteConfig `yaml:"routes"`
}

func Load(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		log.Fatal().Str("path", path).Err(err).Msg("failed to read config file")
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to read config file")
		return nil, err
	}
	return &cfg, nil
}
