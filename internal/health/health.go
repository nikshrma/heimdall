// Package health is the base of the health checking for the load balancer
package health

import (
	"net/http"
	"time"

	"github.com/nikshrma/heimdall/internal/backend"
)

var healthClient = &http.Client{
	Timeout: 3 * time.Second,
}

func CheckHealth(b *backend.Backend) bool {
	url := b.URL.String() + "/health"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := healthClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
