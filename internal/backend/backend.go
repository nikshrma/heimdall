// Package backend exports the runtime backend type
package backend

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"

	"github.com/nikshrma/heimdall/internal/health"
	"github.com/rs/zerolog/log"
)

type Backend struct {
	url   *url.URL
	Proxy *httputil.ReverseProxy

	polling atomic.Bool

	healthy      atomic.Bool
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
	b := &Backend{
		Proxy: proxy,
		url:   target,
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		b.MarkFailure()
		w.WriteHeader(http.StatusBadGateway)
	}
	b.healthy.Store(true)
	return b, nil
}
func (b *Backend) URL() *url.URL { return b.url }

func (b *Backend) IsHealthy() bool {
	return b.healthy.Load()
}

func (b *Backend) MarkSuccess() {
	b.failureCount.Store(0)
	b.successCount.Add(1)
	if b.successCount.Load() >= 3 {
		b.healthy.Store(true)
		log.Info().
			Str("backend", b.URL().String()).
			Msg("backend marked healthy")
	}
}

func (b *Backend) MarkFailure() {
	b.successCount.Store(0)
	b.failureCount.Add(1)
	if b.failureCount.Load() >= 3 {
		b.healthy.Store(false)
		log.Info().
			Str("backend", b.URL().String()).
			Msg("backend marked unhealthy")
		if b.polling.CompareAndSwap(false, true) {
			go health.StartPolling(b)
		}
	}
}

func (b *Backend) StopPolling() {
	b.polling.Store(false)
}
