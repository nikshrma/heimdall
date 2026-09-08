// Package config parses the YAML config
package config

import (
	"errors"
	"os"
	"time"

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
type LimiterConfig struct {
	Enabled    bool          `yaml:"enabled"`
	NumShards  int           `yaml:"shards"`
	Capacity   float64       `yaml:"capacity"`
	RefillRate int64         `yaml:"refillRate"`
	TTL        time.Duration `yaml:"ttl"`
}

type Config struct {
	Log         LogConfig     `yaml:"log"`
	Routes      []RouteConfig `yaml:"routes"`
	LimiterVars LimiterConfig `yaml:"limiter-conf"`
}

func validateLimiterConfig(cfg *LimiterConfig) error {
	if !cfg.Enabled {
		return nil
	}

	if cfg.NumShards <= 0 {
		return errors.New("invalid limiter shards")
	}
	if cfg.Capacity <= 0 {
		return errors.New("invalid limiter capacity")
	}
	if cfg.RefillRate <= 0 {
		return errors.New("invalid limiter refillRate")
	}
	if cfg.TTL <= 0 {
		return errors.New("invalid limiter ttl")
	}

	return nil
}

func Load(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}
	if err := validateLimiterConfig(&cfg.LimiterVars); err != nil {
		return nil, err
	}
	return &cfg, nil
}
