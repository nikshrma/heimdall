// Package backend exports the runtime backend type
package backend

// TODO: take failureCount, successCount, slowThreshold inputs in the config instead of hardCoded vals
import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

type State int32

const (
	Closed State = iota
	HalfOpen
	Open
)

type Backend struct {
	url   *url.URL
	Proxy *httputil.ReverseProxy

	state atomic.Int32

	failureCount  atomic.Int32
	successCount  atomic.Int32
	trialInFlight atomic.Bool

	cooldown      time.Duration
	slowThreshold time.Duration
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
		w.WriteHeader(http.StatusBadGateway)
	}
	proxy.Transport = &breakerTransport{
		next: http.DefaultTransport,
		b:    b,
	}
	b.state.Store(int32(Closed))
	b.cooldown = 10 * time.Second
	b.slowThreshold = 3 * time.Second
	return b, nil
}
func (b *Backend) URL() *url.URL { return b.url }

func (b *Backend) AllowRequest() bool {
	v, err := strconv.ParseBool(os.Getenv("BREAKER_ENABLED"))
	if err != nil {
		v = true
	}
	if !v {
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
		if b.successCount.Load() >= 3 {
			b.state.Store(int32(Closed))
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
		if b.failureCount.Load() >= 3 {
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
	b.successCount.Store(0)
	b.trialInFlight.Store(false)
	time.AfterFunc(b.cooldown, func() {
		b.state.CompareAndSwap(int32(Open), int32(HalfOpen))
	})
}
