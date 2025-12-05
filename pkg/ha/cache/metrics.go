package cache

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	metricsNamespace = "ha"
	metricsSubsystem = "cache"
)

var (
	// Cache hit/miss metrics
	cacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "hits_total",
			Help:      "Total number of cache hits",
		},
		[]string{"cache_type", "level"}, // level: l1, l2, l2_retry
	)

	cacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "misses_total",
			Help:      "Total number of cache misses",
		},
		[]string{"cache_type", "level"},
	)

	cacheInvalidations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "invalidations_total",
			Help:      "Total number of cache invalidations",
		},
		[]string{"cache_type", "source"}, // source: manual, pubsub
	)

	// Chain query metrics
	chainQueries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "chain_queries_total",
			Help:      "Total number of chain queries (cache misses that hit chain)",
		},
		[]string{"query_type"},
	)

	chainQueryErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "chain_query_errors_total",
			Help:      "Total number of chain query errors",
		},
		[]string{"query_type"},
	)

	// chainQueryLatency tracks latency of chain queries (for future use)
	_ = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "chain_query_latency_seconds",
			Help:      "Latency of chain queries",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"query_type"},
	)

	// Session cache specific metrics
	sessionRewardableChecks = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "session_rewardable_checks_total",
			Help:      "Total number of session rewardability checks",
		},
		[]string{"result"}, // result: rewardable, non_rewardable
	)

	sessionMarkedNonRewardable = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "sessions_marked_non_rewardable_total",
			Help:      "Total number of sessions marked as non-rewardable",
		},
		[]string{"reason"},
	)

	// Block event metrics
	blockEventsPublished = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "block_events_published_total",
			Help:      "Total number of block events published",
		},
	)

	blockEventsReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "block_events_received_total",
			Help:      "Total number of block events received",
		},
	)

	currentBlockHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "current_block_height",
			Help:      "Current block height as seen by the cache",
		},
	)

	// lockAcquisitions tracks distributed lock acquisitions (for future use)
	_ = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "lock_acquisitions_total",
			Help:      "Total number of distributed lock acquisitions",
		},
		[]string{"lock_type", "result"}, // result: success, failed
	)
)
