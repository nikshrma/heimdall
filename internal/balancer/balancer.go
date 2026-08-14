// Package balancer exports the balancer with the chosen type of algorithm
package balancer

import (
	"github.com/nikshrma/heimdall/internal/backend"
)

type Balancer interface {
	Next(excluded map[*backend.Backend]struct{}) *backend.Backend
}
