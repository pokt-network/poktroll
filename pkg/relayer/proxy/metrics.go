package proxy

import (
	stdprometheus "github.com/prometheus/client_golang/prometheus"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
)

var (
	// relaysTotal is a Counter metric for the total requests processed by the relay miner.
	// It increments to track proxy requests and is labeled by 'proxy_name' and 'service_id',
	// essential for monitoring load and traffic on different proxies and services.
	//
	// Usage:
	// - Monitor total request load.
	// - Compare requests across services or proxies.
	relaysTotal metrics.Counter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: "relayminer",
		Name:      "requests_total",
		Help:      "Total number of requests processed, labeled by proxy name and service ID.",
	}, []string{"proxy_name", "service_id"})

	// relaysErrorsTotal is a Counter for total error events in the relay miner.
	// It increments with each error, labeled by 'proxy_name' and 'service_id',
	// crucial for pinpointing error-prone areas for reliability improvement.
	//
	// Usage:
	// - Track and analyze error types and distribution.
	// - Compare error rates for reliability analysis.
	relaysErrorsTotal metrics.Counter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: "relayminer",
		Name:      "errors_total",
		Help:      "Total number of error events.",
	}, []string{"proxy_name", "service_id"})

	// relaysDurationSeconds observes request durations in the relay miner.
	// This histogram, labeled by 'proxy_name' and 'service_id', measures response times,
	// vital for performance analysis under different loads.
	//
	// Buckets:
	// - 0.1s to 15s range, capturing response times from very fast to upper limit.
	//
	// Usage:
	// - Analyze typical response times and long-tail latency issues.
	// - Compare performance across services or environments.
	relaysDurationSeconds metrics.Histogram = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Subsystem: "relayminer",
		Name:      "request_duration_seconds",
		Help:      "Histogram of request durations for performance analysis.",
		Buckets:   []float64{0.1, 0.5, 1, 2, 5, 15},
	}, []string{"proxy_name", "service_id"})

	// TODO(@okdas): add `response_size_bytes`. Postponing to avoid new HTTP server writer implementation;
	// will reevaluate with historical load data.
	// responseSizeBytes is a histogram metric for observing proxy response size distribution.
	// It counts responses in bytes, with buckets:
	// - 100 bytes to 50,000 bytes, capturing a range from small to large responses.
	// This data helps in accurately representing response size distribution and is vital
	// for performance tuning.
	//
	// Note: Consider configuring bucket sizes externally for flexible adjustments
	// in response to different data patterns or deployment scenarios.
	// responseSizeBytes metrics.Histogram = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
	// 	Subsystem: "relayminer",
	// 	Name:      "response_size_bytes",
	// 	Buckets:   []float64{100, 500, 1000, 5000, 10000, 50000},
	// }, []string{"proxy_name", "service_id"})
)
