// Package health is the base of the health checking for the load balancer
package health

import (
	"net/http"
	"net/url"
	"time"
)

type Checker interface {
	URL() *url.URL
	MarkSuccess()
	MarkFailure()
	IsHealthy() bool
	StopPolling()
}

var healthClient = &http.Client{
	Timeout: 3 * time.Second,
}

func StartPolling(b Checker) {
	defer b.StopPolling()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		if b.IsHealthy() {
			return
		}
		if CheckHealth(b) {
			b.MarkSuccess()
			if b.IsHealthy() {
				return
			}
		} else {
			b.MarkFailure()
		}
	}
}

func CheckHealth(b Checker) bool {
	url := b.URL().String() + "/health"
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
