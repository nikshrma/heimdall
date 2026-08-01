package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	err := os.WriteFile(configPath, []byte(`
log:
  level: debug
routes:
  - path: /users
    methods:
      - POST
    stripPrefix: true
    backends:
      - http://im1:3001
      - http://im2:3002
      - http://im3:3003
  - path: /contacts
    methods:
      - GET
    stripPrefix: false
    backends:
      - http://im4:3004
      - http://im5:3005
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Log.Level != "debug" {
		t.Fatalf("expected debug, got %q", cfg.Log.Level)
	}

	if len(cfg.Routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(cfg.Routes))
	}

	if cfg.Routes[0].Path != "/users" {
		t.Errorf("expected /users, got %q", cfg.Routes[0].Path)
	}

	if !cfg.Routes[0].StripPrefix {
		t.Error("expected StripPrefix to be true")
	}

	if len(cfg.Routes[0].Backends) != 3 {
		t.Errorf("expected 3 backends, got %d", len(cfg.Routes[0].Backends))
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("does-not-exist.yml")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	// Fact: 0o is enforced because using a stand alone leading zero is deprecated and might lead to bugs
	os.WriteFile(path, []byte(": invalid"), 0o644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected yaml error")
	}
}
