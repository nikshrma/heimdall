// Package backend exports the runtime backend type
package backend

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nikshrma/heimdall/internal/health"
)

type Backend struct {
	URL   *url.URL
	Proxy *httputil.ReverseProxy

	mu      sync.RWMutex
	polling atomic.Bool

	healthy      bool
	failureCount atomic.Int32
	successCount atomic.Int32
}

func New(be string) (*Backend, error) {
	target, err := url.Parse(be)
	if err != nil {
		return nil, err
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()
		},
	}
	return &Backend{
		Proxy: proxy,
		URL:   target,
	}, nil
}

func IsHealthy(b *Backend) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.healthy
}

func MarkSuccess(b *Backend) {
	b.failureCount.Store(0)
	b.successCount.Add(1)
	if b.successCount.Load() >= 3 {
		b.healthy = true
	}
}

func MarkFailure(b *Backend) {
	b.successCount.Store(0)
	b.failureCount.Add(1)
	if b.failureCount.Load() >= 3 {
		b.healthy = false
		if b.polling.CompareAndSwap(false, true) {
			go StartPolling(b)
		}
		defer b.polling.Store(false)
	}
}

func StartPolling(b *Backend) {
	defer b.polling.Store(false)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		if health.CheckHealth(b) {
			MarkSuccess(b)
			if IsHealthy(b) {
				return
			}
		} else {
			MarkSuccess(b)
		}
	}
}
