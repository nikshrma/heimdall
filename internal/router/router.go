// Package router contains the runtime route structs parsed from the YAML
package router

import (
	"net/http/httputil"
	"net/url"

	"github.com/nikshrma/heimdall/internal/balancer"
)

type Backend struct {
	URL   *url.URL
	proxy httputil.ReverseProxy
}
type Routes struct {
	Path        string
	Methods     []string
	StripPrefix bool
	bala        balancer.Balancer
	backends    []Backend
}
