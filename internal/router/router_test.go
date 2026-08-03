package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikshrma/heimdall/internal/config"
)

func TestBuildRoutes(t *testing.T) {
	cfg := config.Config{
		Routes: []config.RouteConfig{
			{
				Path:     "/api",
				Methods:  []string{http.MethodGet},
				Backends: []string{"http://localhost:8080"},
			},
		},
	}
	built, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(built) != 1 {
		t.Error("expected length 1")
	}
	if built[0].Path != "/api" {
		t.Error("expected /api endpoint")
	}
}

func TestMatch(t *testing.T) {
	routes, _ := Build(config.Config{
		Routes: []config.RouteConfig{
			{
				Path:     "/api",
				Backends: []string{"http://localhost:8080"},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)

	route, err := Match(routes, req)
	if err != nil {
		t.Fatal(err)
	}

	if route == nil {
		t.Fatal("expected matching route")
	}
}

func TestMethodNotAllowed(t *testing.T) {
	routes, _ := Build(config.Config{
		Routes: []config.RouteConfig{
			{
				Path:     "/api",
				Methods:  []string{http.MethodPost},
				Backends: []string{"http://localhost:8080"},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)

	_, err := Match(routes, req)
	if err != ErrorMethodNotAllowed {
		t.Fatalf("expected %v, got %v", ErrorMethodNotAllowed, err)
	}
}

func TestMethodMatch(t *testing.T) {
	if !MethodMatch(nil, http.MethodGet) {
		t.Fatal("expected nil methods to match")
	}

	if !MethodMatch([]string{http.MethodGet}, http.MethodGet) {
		t.Fatal("expected GET to match")
	}

	if MethodMatch([]string{http.MethodPost}, http.MethodGet) {
		t.Fatal("expected GET not to match")
	}
}
