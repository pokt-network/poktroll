package query

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Query metrics (reserved for future instrumentation)
var (
	_ = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ha",
			Subsystem: "query",
			Name:      "queries_total",
			Help:      "Total number of chain queries",
		},
		[]string{"client", "method"},
	)

	_ = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ha",
			Subsystem: "query",
			Name:      "query_errors_total",
			Help:      "Total number of query errors",
		},
		[]string{"client", "method"},
	)

	_ = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ha",
			Subsystem: "query",
			Name:      "query_latency_seconds",
			Help:      "Query latency in seconds",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"client", "method"},
	)

	// Cache metrics (reserved for future instrumentation)
	_ = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ha",
			Subsystem: "query",
			Name:      "cache_hits_total",
			Help:      "Total number of cache hits",
		},
		[]string{"client", "cache_type"},
	)

	_ = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ha",
			Subsystem: "query",
			Name:      "cache_misses_total",
			Help:      "Total number of cache misses",
		},
		[]string{"client", "cache_type"},
	)

	_ = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ha",
			Subsystem: "query",
			Name:      "cache_size",
			Help:      "Current cache size (number of entries)",
		},
		[]string{"client", "cache_type"},
	)
)
