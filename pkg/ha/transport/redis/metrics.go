package redis

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	metricsNamespace = "ha"
	metricsSubsystem = "transport_redis"
)

var (
	// Publisher metrics

	publishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "published_total",
			Help:      "Total number of mined relays published to Redis Streams",
		},
		[]string{"supplier_addr", "service_id"},
	)

	publishErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "publish_errors_total",
			Help:      "Total number of publish errors",
		},
		[]string{"supplier_addr", "service_id"},
	)

	// publishLatency reserved for future instrumentation
	_ = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "publish_latency_seconds",
			Help:      "Latency of publish operations",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"supplier_addr"},
	)

	// Consumer metrics

	consumedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "consumed_total",
			Help:      "Total number of mined relays consumed from Redis Streams",
		},
		[]string{"supplier_addr", "service_id"},
	)

	consumeErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "consume_errors_total",
			Help:      "Total number of consume errors",
		},
		[]string{"supplier_addr", "error_type"},
	)

	ackedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "acked_total",
			Help:      "Total number of messages acknowledged",
		},
		[]string{"supplier_addr"},
	)

	pendingMessages = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "pending_messages",
			Help:      "Current number of pending (unacknowledged) messages",
		},
		[]string{"supplier_addr"},
	)

	// consumerLag reserved for future instrumentation
	_ = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "consumer_lag",
			Help:      "Consumer lag (messages behind head of stream)",
		},
		[]string{"supplier_addr"},
	)

	// streamLength reserved for future instrumentation
	_ = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "stream_length",
			Help:      "Current length of Redis stream",
		},
		[]string{"supplier_addr"},
	)

	claimedMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "claimed_total",
			Help:      "Total number of messages claimed from idle consumers",
		},
		[]string{"supplier_addr"},
	)

	deserializationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "deserialization_errors_total",
			Help:      "Total number of message deserialization errors",
		},
		[]string{"supplier_addr"},
	)

	// End-to-end latency from publish to consume
	endToEndLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "end_to_end_latency_seconds",
			Help:      "End-to-end latency from publish to consume",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"supplier_addr", "service_id"},
	)
)
