// Package config parses the YAML config
package config

type RouteConfig struct {
	Path        string   `yaml:"path"`
	Methods     []string `yaml:"methods"`
	StripPrefix bool     `yaml:"stripPrefix"`
	Backends    []string `yaml:"backends"`
}

type Config struct {
	Routes []RouteConfig `yaml:"routes"`
}
