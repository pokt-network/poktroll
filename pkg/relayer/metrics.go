package relayer

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	relayMinerProcess = "relayminer"

	requestsTotal          = "requests_total"
	requestsErrorsTotal    = "requests_errors_total"
	requestsSuccessTotal   = "requests_success_total"
	requestSizeBytes       = "request_size_bytes"
	responseSizeBytes      = "response_size_bytes"
	smtSizeBytes           = "smt_size_bytes"
	relayDurationSeconds   = "relay_duration_seconds"
	serviceDurationSeconds = "service_duration_seconds"
)

var (
	defaultBuckets = []float64{
		// Sub-50ms (cache hits, internal optimization, fast responses, potential internal errors, etc.)
		0.01, 0.05,
		// Primary range: 50ms to 1s (majority of traffic, normal responses, etc...)
		0.1, 0.2, 0.4, 0.5, 0.75, 1.0,
		// Long tail: > 1s (slow queries, rollovers, cold state, failed, etc.)
		2.0, 5.0, 10.0, 30.0,
	}
)

var (
	// RelaysTotal is a Counter metric for the total requests processed by the relay miner.
	// It increments to track proxy requests and is labeled by 'service_id',
	// essential for monitoring load and traffic on different proxies and services.
	//
	// Usage:
	// - Monitor total request load.
	// - Compare requests across services or proxies.
	RelaysTotal = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: relayMinerProcess,
		Name:      requestsTotal,
		Help:      "Total number of requests processed, labeled by service ID.",
	}, []string{"service_id", "supplier_operator_address"})

	// RelaysErrorsTotal is a Counter for total error events in the relay miner.
	// It increments with each error, labeled by 'service_id',
	// crucial for pinpointing error-prone areas for reliability improvement.
	//
	// Usage:
	// - Track and analyze error types and distribution.
	// - Compare error rates for reliability analysis.
	RelaysErrorsTotal = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: relayMinerProcess,
		Name:      requestsErrorsTotal,
		Help:      "Total number of error events.",
	}, []string{"service_id"})

	// RelaysSuccessTotal is a Counter metric for successful requests in the relay miner.
	// It increments with each successful request, labeled by 'service_id'.
	RelaysSuccessTotal = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: relayMinerProcess,
		Name:      requestsSuccessTotal,
		Help:      "Total number of successful requests processed, labeled by service ID.",
	}, []string{"service_id"})

	// RelaysDurationSeconds observes the end-to-end duration of the request in the relay miner.
	//
	// It includes the duration of the request to the backend service/data node AS WELL AS
	// the duration of the RelayMiner's internal overhead.
	//
	// Usage:
	// - Analyze typical response times and long-tail latency issues.
	// - Compare performance across services or environments.
	RelaysDurationSeconds = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Subsystem: relayMinerProcess,
		Name:      relayDurationSeconds,
		Help:      "Histogram of request durations for performance analysis.",
		Buckets:   defaultBuckets,
	}, []string{"service_id", "status_code"})

	// ServiceDurationSeconds observes the duration of the request to the backend service/data node outside of the RelayMiner.
	//
	// This histogram, labeled by 'service_id' and 'status_code', measures response times,
	// vital for performance analysis under different loads.
	//
	// It is a complementary metric to RelaysDurationSeconds allowing to isolate the
	// performance of the data node and the overhead of the RelayMiner's internal processing.
	//
	// Usage:
	// - Analyze typical response times and long-tail latency issues.
	// - Compare performance across services or environments.
	ServiceDurationSeconds = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Subsystem: relayMinerProcess,
		Name:      serviceDurationSeconds,
		Help:      "Histogram of service call durations for performance analysis.",
		Buckets:   defaultBuckets,
	}, []string{"service_id", "status_code"})

	// RelayResponseSizeBytes is a histogram metric for observing response size distribution.
	// It counts responses in bytes, with buckets:
	// - 100 bytes to 50,000 bytes, capturing a range from small to large responses.
	// This data helps in accurately representing response size distribution and is vital
	// for performance tuning.
	//
	// TODO_TECHDEBT: Consider configuring bucket sizes externally for flexible adjustments
	// in response to different data patterns or deployment scenarios.
	RelayResponseSizeBytes = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Subsystem: relayMinerProcess,
		Name:      responseSizeBytes,
		Help:      "Histogram of response sizes in bytes for performance analysis.",
		Buckets:   []float64{100, 500, 1000, 5000, 10000, 50000},
	}, []string{"service_id"})

	// RelayRequestSizeBytes is a histogram metric for observing request size distribution.
	// It counts requests in bytes, with buckets:
	// - 100 bytes to 50,000 bytes, capturing a range from small to large requests.
	// This data helps in accurately representing request size distribution and is vital
	// for performance tuning.
	RelayRequestSizeBytes = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Subsystem: relayMinerProcess,
		Name:      requestSizeBytes,
		Help:      "Histogram of request sizes in bytes for performance analysis.",
		Buckets:   []float64{100, 500, 1000, 5000, 10000, 50000},
	}, []string{"service_id"})
)

// CaptureRelayDuration records the internal end-to-end duration of handling a relay which includes
// the network call to the backend data / service node.
// It calculates the elapsed time since the RelayMiner started processing the request BEFORE doing all of its internal processing.
func CaptureRelayDuration(serviceId string, startTime time.Time, statusCode int) {
	duration := time.Since(startTime).Seconds()

	RelaysDurationSeconds.
		With("service_id", serviceId).
		With("status_code", fmt.Sprintf("%d", statusCode)).
		Observe(duration)
}

// CaptureServiceDuration records the duration of a request to the backend data / service node explicitly.
// It is labeled by service ID and HTTP status code.
// It calculates the elapsed time since the RelayMiner started the outbound network call AFTER doing all of its internal processing.
func CaptureServiceDuration(serviceId string, startTime time.Time, statusCode int) {
	duration := time.Since(startTime).Seconds()

	ServiceDurationSeconds.
		With("service_id", serviceId).
		With("status_code", fmt.Sprintf("%d", statusCode)).
		Observe(duration)
}
