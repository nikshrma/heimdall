// Package balancer exports the balancer with the chosen type of algorithm
package balancer

import (
	"github.com/nikshrma/heimdall/internal/backend"
)

type Balancer interface {
	Next() *backend.Backend
}
