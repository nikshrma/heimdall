// Package router contains the runtime route structs parsed from the YAML
package router

import (
	"errors"
	"net/http"
	"slices"

	"github.com/nikshrma/heimdall/internal/backend"
	"github.com/nikshrma/heimdall/internal/balancer"
	"github.com/nikshrma/heimdall/internal/config"
)

var ErrorMethodNotAllowed = errors.New("method not allowed")

type Route struct {
	Path        string
	Methods     []string
	StripPrefix bool
	Balancer    balancer.Balancer
	Backends    []*backend.Backend
}

type Matcher struct {
	root *radixNode
}

type radixNode struct {
	prefix   string
	route    *Route
	children []*radixNode
}

func buildBackends(backends []string, cfg config.BreakerConfig) ([]*backend.Backend, error) {
	var runtimeBackends []*backend.Backend
	breakerCfg := backend.BreakerConfig{
		Enabled:          cfg.Enabled,
		FailureThreshold: cfg.FailureThreshold,
		SuccessThreshold: cfg.SuccessThreshold,
		Cooldown:         cfg.Cooldown,
		SlowThreshold:    cfg.SlowThreshold,
	}
	for _, be := range backends {
		b, err := backend.NewWithConfig(be, breakerCfg)
		if err != nil {
			return nil, err
		}
		runtimeBackends = append(runtimeBackends, b)
	}
	return runtimeBackends, nil
}

func Build(cfg config.Config) ([]*Route, error) {
	var runtimeRoutes []*Route
	for _, rc := range cfg.Routes {
		backends, err := buildBackends(rc.Backends, cfg.BreakerVars)
		if err != nil {
			return nil, err
		}

		runtimeRoutes = append(runtimeRoutes, &Route{
			Path:        rc.Path,
			Methods:     rc.Methods,
			StripPrefix: rc.StripPrefix,
			Backends:    backends,
			Balancer:    balancer.NewRoundRobin(backends),
		})
	}
	if len(runtimeRoutes) == 0 {
		return nil, errors.New("no routes configured")
	}
	return runtimeRoutes, nil
}

func NewMatcher(routes []*Route) *Matcher {
	m := &Matcher{
		root: &radixNode{},
	}
	for _, route := range routes {
		m.root.insert(route.Path, route)
	}
	return m
}

func Match(routes []*Route, req *http.Request) (*Route, error) {
	return NewMatcher(routes).Match(req)
}

func (m *Matcher) Match(req *http.Request) (*Route, error) {
	var best *Route
	methodMismatch := false

	for _, route := range m.root.match(req.URL.Path) {
		if MethodMatch(route.Methods, req.Method) {
			best = route
			methodMismatch = false
		} else if best == nil {
			methodMismatch = true
		}
	}

	if best != nil {
		return best, nil
	}
	if methodMismatch {
		return nil, ErrorMethodNotAllowed
	}
	return nil, nil
}

func (n *radixNode) insert(path string, route *Route) {
	if path == "" {
		n.route = route
		return
	}

	for _, child := range n.children {
		common := commonPrefix(path, child.prefix)
		if common == 0 {
			continue
		}

		if common == len(child.prefix) {
			child.insert(path[common:], route)
			return
		}

		split := &radixNode{
			prefix:   child.prefix[common:],
			route:    child.route,
			children: child.children,
		}
		child.prefix = child.prefix[:common]
		child.route = nil
		child.children = []*radixNode{split}

		if common == len(path) {
			child.route = route
			return
		}

		child.children = append(child.children, &radixNode{
			prefix: path[common:],
			route:  route,
		})
		return
	}

	n.children = append(n.children, &radixNode{
		prefix: path,
		route:  route,
	})
}

func (n *radixNode) match(path string) []*Route {
	var matches []*Route
	node := n

	for {
		if node.route != nil {
			matches = append(matches, node.route)
		}
		if path == "" {
			return matches
		}

		var next *radixNode
		for _, child := range node.children {
			if hasPrefix(path, child.prefix) {
				next = child
				break
			}
		}
		if next == nil {
			return matches
		}

		path = path[len(next.prefix):]
		node = next
	}
}

func commonPrefix(a, b string) int {
	max := min(len(a), len(b))
	for i := 0; i < max; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return max
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func MethodMatch(methods []string, method string) bool {
	if len(methods) == 0 {
		return true // no restriction configured
	}
	return slices.Contains(methods, method)
}
