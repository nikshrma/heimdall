// Package backend exports the runtime backend type
package backend

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/nikshrma/heimdall/internal/metrics"
	"github.com/rs/zerolog/log"
)

type State int32

const (
	Closed State = iota
	HalfOpen
	Open
)

type BreakerConfig struct {
	FailureThreshold int32
	SuccessThreshold int32
	Cooldown         time.Duration
	SlowThreshold    time.Duration
}

var DefaultBreakerConfig = BreakerConfig{
	FailureThreshold: 3,
	SuccessThreshold: 3,
	Cooldown:         10 * time.Second,
	SlowThreshold:    3 * time.Second,
}

type Backend struct {
	url   *url.URL
	Proxy *httputil.ReverseProxy

	state atomic.Int32

	failureCount  atomic.Int32
	successCount  atomic.Int32
	trialInFlight atomic.Bool

	failureThreshold int32
	successThreshold int32
	cooldown         time.Duration
	slowThreshold    time.Duration
	enabled          bool
}

func New(be string) (*Backend, error) {
	return NewWithConfig(be, DefaultBreakerConfig)
}

func NewWithConfig(be string, cfg BreakerConfig) (*Backend, error) {
	cfg = cfg.withDefaults()
	target, err := url.Parse(be)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()
		},
	}
	v, err := strconv.ParseBool(os.Getenv("BREAKER_ENABLED"))
	if err != nil {
		v = true
	}
	b := &Backend{
		Proxy:            proxy,
		url:              target,
		failureThreshold: cfg.FailureThreshold,
		successThreshold: cfg.SuccessThreshold,
		cooldown:         cfg.Cooldown,
		slowThreshold:    cfg.SlowThreshold,
		enabled:          v,
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusBadGateway)
	}
	proxy.Transport = &breakerTransport{
		next: transport,
		b:    b,
	}
	b.state.Store(int32(Closed))
	return b, nil
}
func (b *Backend) URL() *url.URL { return b.url }

func (cfg BreakerConfig) withDefaults() BreakerConfig {
	defaults := DefaultBreakerConfig
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = defaults.FailureThreshold
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = defaults.SuccessThreshold
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = defaults.Cooldown
	}
	if cfg.SlowThreshold <= 0 {
		cfg.SlowThreshold = defaults.SlowThreshold
	}
	return cfg
}

func (b *Backend) AllowRequest() bool {
	if !b.enabled {
		return true
	}
	state := b.state.Load()
	switch state {
	case int32(Closed):
		return true
	case int32(Open):
		return false
	case int32(HalfOpen):
		return b.trialInFlight.CompareAndSwap(false, true)
	default:
		return false
	}
}

func (b *Backend) MarkSuccess() {
	state := b.state.Load()
	switch state {
	case int32(Closed):
		b.failureCount.Store(0)
	case int32(HalfOpen):
		b.successCount.Add(1)
		if b.successCount.Load() >= b.successThreshold {
			b.state.Store(int32(Closed))
			metrics.CircuitBreakerState.WithLabelValues(b.URL().String()).Set(float64(Closed))
			b.successCount.Store(0)
			b.failureCount.Store(0)
		}
	}
}

func (b *Backend) MarkFailure() {
	b.successCount.Store(0)
	state := b.state.Load()
	switch state {
	case int32(Closed):
		b.failureCount.Add(1)
		if b.failureCount.Load() >= b.failureThreshold {
			log.Info().
				Str("backend", b.URL().String()).
				Msg("backend marked open")
			b.trip()
		}
	case int32(HalfOpen):
		b.trip()
	}
}

func (b *Backend) trip() {
	b.state.Store(int32(Open))
	metrics.CircuitBreakerTripsTotal.WithLabelValues(b.URL().String()).Inc()
	metrics.CircuitBreakerState.WithLabelValues(b.URL().String()).Set(float64(Open))
	b.successCount.Store(0)
	b.trialInFlight.Store(false)
	time.AfterFunc(b.cooldown, func() {
		if b.state.CompareAndSwap(int32(Open), int32(HalfOpen)) {
			metrics.CircuitBreakerState.WithLabelValues(b.URL().String()).Set(float64(HalfOpen))
		}
	})
}
