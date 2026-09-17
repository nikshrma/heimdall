// Package config parses the YAML config
package config

import (
	"errors"
	"os"
	"strconv"
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
	Enabled     bool          `yaml:"enabled"`
	NumShards   int           `yaml:"shards"`
	Capacity    float64       `yaml:"capacity"`
	RefillRate  int64         `yaml:"refillRate"`
	TTL         time.Duration `yaml:"ttl"`
	CleanUpTime time.Duration `yaml:"cleanUpTime"`
}
type BreakerConfig struct {
	Enabled          bool          `yaml:"enabled"`
	FailureThreshold int32         `yaml:"failureThreshold"`
	SuccessThreshold int32         `yaml:"successThreshold"`
	Cooldown         time.Duration `yaml:"cooldown"`
	SlowThreshold    time.Duration `yaml:"slowThreshold"`
}

type Config struct {
	Log         LogConfig     `yaml:"log"`
	Routes      []RouteConfig `yaml:"routes"`
	LimiterVars LimiterConfig `yaml:"limiter-conf"`
	BreakerVars BreakerConfig `yaml:"breaker-conf"`
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
	if cfg.CleanUpTime <= 0 {
		return errors.New("invalid clean up duration")
	}

	return nil
}

func validateBreakerConfig(cfg *BreakerConfig) error {
	if !cfg.Enabled {
		return nil
	}

	if cfg.FailureThreshold <= 0 {
		return errors.New("invalid breaker failureThreshold")
	}
	if cfg.SuccessThreshold <= 0 {
		return errors.New("invalid breaker successThreshold")
	}
	if cfg.Cooldown <= 0 {
		return errors.New("invalid breaker cooldown")
	}
	if cfg.SlowThreshold <= 0 {
		return errors.New("invalid breaker slowThreshold")
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

	if envVal := os.Getenv("BREAKER_ENABLED"); envVal != "" {
		if val, err := strconv.ParseBool(envVal); err == nil {
			cfg.BreakerVars.Enabled = val
		}
	}
	if envVal := os.Getenv("RATE_LIMIT_ENABLED"); envVal != "" {
		if val, err := strconv.ParseBool(envVal); err == nil {
			cfg.LimiterVars.Enabled = val
		}
	}

	if err := validateLimiterConfig(&cfg.LimiterVars); err != nil {
		return nil, err
	}
	if err := validateBreakerConfig(&cfg.BreakerVars); err != nil {
		return nil, err
	}
	return &cfg, nil
}
