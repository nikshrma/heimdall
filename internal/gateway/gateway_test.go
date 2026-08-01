package gateway

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nikshrma/heimdall/internal/backend"
	"github.com/nikshrma/heimdall/internal/balancer"
	"github.com/nikshrma/heimdall/internal/router"
)

func newTestBackend(t *testing.T, handler http.HandlerFunc) (*backend.Backend, error) {
	t.Helper()

	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	return backend.New(u.String())
}

func TestGatewayNotFound(t *testing.T) {
	g := New(nil)

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rr := httptest.NewRecorder()

	g.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestGatewayProxyRequest(t *testing.T) {
	be, _ := newTestBackend(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users" {
			t.Fatalf("expected /api/users, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})

	route := &router.Route{
		Path:     "/api",
		Methods:  []string{http.MethodGet},
		Backends: []*backend.Backend{be},
		Balancer: balancer.NewRoundRobin([]*backend.Backend{be}),
	}

	g := New([]*router.Route{route})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rr := httptest.NewRecorder()

	g.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestGatewayStripPrefix(t *testing.T) {
	be, _ := newTestBackend(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users" {
			t.Fatalf("expected /users, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})

	route := &router.Route{
		Path:        "/api",
		StripPrefix: true,
		Methods:     []string{http.MethodGet},
		Backends:    []*backend.Backend{be},
		Balancer:    balancer.NewRoundRobin([]*backend.Backend{be}),
	}

	g := New([]*router.Route{route})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rr := httptest.NewRecorder()

	g.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
