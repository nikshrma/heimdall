// Package metrics makes all necessary metrics that heimdall collects for monitoring
package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	TotalRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests handled",
		}, []string{"route", "method", "status"},
	)
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests in seconds",
			Buckets: []float64{
				0.001, 0.005, 0.01, 0.025, 0.05,
				0.1, 0.25, 0.5, 1, 2, 5,
			},
		}, []string{"route", "method"},
	)
	InFlightRequests = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Total HTTP requests being processed",
		}, []string{"route"},
	)
	ProxyBackendRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxy_backend_requests_total",
			Help: "Number of HTTP requests made to proxy backend",
		}, []string{"backend", "route", "method", "status"},
	)
	ProxyBackendRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "proxy_backend_request_duration_seconds",
			Help: "Duration of HTTP requests made to Proxy backends",
			Buckets: []float64{
				0.001, 0.005, 0.01, 0.025, 0.05,
				0.1, 0.25, 0.5, 1, 2, 5,
			},
		}, []string{"backend", "route"},
	)
	ProxyBackendInFlightRequests = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "proxy_backend_in_flight_requests",
			Help: "Number of requests being processed by proxy backends",
		}, []string{"backend"},
	)
	ProxyBackendSelectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxy_backend_selected_total",
			Help: "Number of times a proxy backend is selected",
		}, []string{"backend", "route"},
	)
	CircuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Current circuit breaker state (0 = closed, 1 = half-open, 2 = open).",
		},
		[]string{"backend"},
	)
	CircuitBreakerTripsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_trips_total",
			Help: "Total number of times the circuit breaker has opened.",
		},
		[]string{"backend"},
	)
	CircuitBreakerRejectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_rejections_total",
			Help: "Total number of requests rejected by the circuit breaker (open or half-open trial in flight).",
		},
		[]string{"backend"},
	)
	ProxyRetriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxy_retries_total",
			Help: "Total number of retry attempts made by the proxy.",
		},
		[]string{"backend", "route", "reason"},
	)
	ProxyRequestsExhaustedRetriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxy_requests_exhausted_retries_total",
			Help: "Total number of requests that failed after exhausting all retry attempts.",
		},
		[]string{"route"},
	)
	RateLimitRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_requests_total",
			Help: "Total number of requests processed by the rate limiter.",
		},
		[]string{"result"},
	)
	RateLimitActiveBuckets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "rate_limit_active_buckets",
			Help: "Number of active IP token buckets currently tracked by the rate limiter.",
		},
	)
)
