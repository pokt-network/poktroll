package relayer

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	relayMinerProcess = "relayminer"

	requestsTotal                              = "requests_total"
	requestsErrorsTotal                        = "requests_errors_total"
	requestsSuccessTotal                       = "requests_success_total"
	requestSizeBytes                           = "request_size_bytes"
	responseSizeBytes                          = "response_size_bytes"
	relayDurationSeconds                       = "relay_duration_seconds"
	serviceDurationSeconds                     = "service_duration_seconds"
	requestPreparationDurationSeconds          = "relay_request_preparation_duration_seconds"
	responsePreparationDurationSeconds         = "relay_response_preparation_duration_seconds"
	fullNodeGRPCCallDurationSeconds            = "full_node_grpc_call_duration_seconds"
	delayedRelayRequestValidationTotal         = "delayed_relay_request_validation_total"
	delayedRelayRequestValidationFailuresTotal = "delayed_relay_request_validation_failures_total"
	delayedRelayRequestRateLimitingCheckTotal  = "delayed_relay_request_rate_limiting_check_total"
	instructionTimeSeconds                     = "instruction_time_seconds"
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

	// TODO_TECHDEBT: Remove those metrics once the session rollover issue is resolved.
	// RelayRequestPreparationDurationSeconds observes the duration of preparing
	// a relay request from an HTTP request.
	//
	// This histogram, labeled by 'service_id', measures the time taken to prepare
	// a relay request, which includes unmarshaling the request body and preparing
	// the RelayRequest structure.
	// Usage:
	// - Analyze the time taken to prepare relay requests.
	// - Identify potential bottlenecks in request preparation.
	RelayRequestPreparationDurationSeconds = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Subsystem: relayMinerProcess,
		Name:      requestPreparationDurationSeconds,
		Help:      "Histogram of relay request preparation durations in seconds.",
		Buckets:   defaultBuckets,
	}, []string{"service_id"})

	// TODO_TECHDEBT: Remove those metrics once the session rollover issue is resolved.
	// RelayResponsePreparationDurationSeconds observes the duration of preparing
	// a relay response from the the service's HTTP response.
	//
	// This histogram, labeled by 'service_id', measures the time taken to prepare
	// a relay response, which includes marshaling the response body and preparing
	// the RelayResponse structure.
	// Usage:
	// - Analyze the time taken to prepare relay responses.
	// - Identify potential bottlenecks in response preparation.
	RelayResponsePreparationDurationSeconds = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Subsystem: relayMinerProcess,
		Name:      responsePreparationDurationSeconds,
		Help:      "Histogram of relay response preparation durations in seconds.",
		Buckets:   defaultBuckets,
	}, []string{"service_id"})

	// FullNodeGRPCCallDurationSeconds is a histogram metric for measuring the duration of gRPC calls.
	//
	// It is labeled by 'component' (e.g., miner, proxy) and 'method', capturing the time taken for each call.
	// This metric is essential for performance monitoring and analysis of gRPC interactions.
	// Usage:
	// - Monitor gRPC call performance.
	// - Identify slow or problematic calls.
	FullNodeGRPCCallDurationSeconds = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Subsystem: relayMinerProcess,
		Name:      fullNodeGRPCCallDurationSeconds,
		Help:      "Histogram of gRPC call durations for performance analysis.",
		Buckets:   defaultBuckets,
	}, []string{"component", "method"})

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

	// DelayedRelayRequestValidationTotal is a Counter metric for tracking delayed validation occurrences.
	// It increments when relay requests are validated after being served (late validation)
	// rather than being validated upfront. This indicates sessions that weren't known at
	// request time and required deferred validation.
	DelayedRelayRequestValidationTotal = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: relayMinerProcess,
		Name:      delayedRelayRequestValidationTotal,
		Help:      "Total number of delayed validation occurrences, labeled by service ID.",
	}, []string{"service_id", "supplier_operator_address"})

	// DelayedRelayRequestValidationFailuresTotal is a Counter metric for tracking delayed validation failures.
	// It increments when relay requests fail validation during the delayed validation process.
	// This typically occurs when the relay request signature is invalid or session validation
	// fails after the relay has already been served.
	DelayedRelayRequestValidationFailuresTotal = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: relayMinerProcess,
		Name:      delayedRelayRequestValidationFailuresTotal,
		Help:      "Total number of delayed validation failures, labeled by service ID.",
	}, []string{"service_id", "supplier_operator_address"})

	// DelayedRelayRequestRateLimitingCheckTotal is a Counter metric for tracking rate limiting during delayed validation.
	// It increments when, during late (deferred) validation, the application is found to have
	// exceeded its allocated stake and the relay is rate limited.
	//
	// User/payment impact:
	// - The relay becomes reward-ineligible (no on-chain payment to the supplier for this relay).
	// - Signals potential fee leakage and helps rate-limit thresholds.
	DelayedRelayRequestRateLimitingCheckTotal = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: relayMinerProcess,
		Name:      delayedRelayRequestRateLimitingCheckTotal,
		Help:      "Total number of delayed validation rate limiting events, labeled by service ID.",
	}, []string{"service_id", "supplier_operator_address"})

	// InstructionTimeSeconds is a Histogram metric for tracking the duration of individual
	// instructions during relay processing. It measures the time between consecutive
	// instruction steps to identify performance bottlenecks and optimize relay handling.
	// The metric is labeled by instruction name to provide granular timing analysis.
	InstructionTimeSeconds = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Subsystem: relayMinerProcess,
		Name:      instructionTimeSeconds,
		Help:      "Histogram of request instruction times in milliseconds for performance analysis.",
		Buckets:   defaultBuckets,
	}, []string{"instruction"})
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

// CaptureRequestPreparationDuration records the duration of the request reading and preparation
// before sending it to the backend service/data node.
// It is labeled by service ID and captures the time taken to process the request.
func CaptureRequestPreparationDuration(serviceId string, startTime time.Time) {
	duration := time.Since(startTime).Seconds()

	RelayRequestPreparationDurationSeconds.
		With("service_id", serviceId).
		Observe(duration)
}

// CaptureResponsePreparationDuration records the duration of preparing the response
// after receiving it from the backend service/data node.
// It is labeled by service ID and captures the time taken to process the response.
func CaptureResponsePreparationDuration(serviceId string, startTime time.Time) {
	duration := time.Since(startTime).Seconds()

	RelayResponsePreparationDurationSeconds.
		With("service_id", serviceId).
		Observe(duration)
}

// CaptureGRPCCallDuration records the duration of a gRPC call.
// It is labeled by component (e.g., miner, proxy) and method, capturing the time taken for the call.
func CaptureGRPCCallDuration(component, method string, startTime time.Time) {
	duration := time.Since(startTime).Seconds()

	FullNodeGRPCCallDurationSeconds.
		With("component", component).
		With("method", method).
		Observe(duration)
}

// CaptureDelayedRelayRequestValidationFailure records a delayed validation failure event.
// This metric is incremented when a relay request fails validation during the delayed
// validation process, typically due to invalid signature or session validation failure
// after the relay has already been served.
func CaptureDelayedRelayRequestValidationFailure(serviceId string, supplierOperatorAddress string) {
	DelayedRelayRequestValidationFailuresTotal.
		With("service_id", serviceId, "supplier_operator_address", supplierOperatorAddress).
		Add(1)
}

// CaptureDelayedRelayRequestValidation records a delayed validation occurrence.
// This metric is incremented when relay requests are validated after being served
// (late validation) rather than being validated upfront. This indicates sessions
// that weren't known at request time and required deferred validation.
func CaptureDelayedRelayRequestValidation(serviceId string, supplierOperatorAddress string) {
	DelayedRelayRequestValidationTotal.
		With("service_id", serviceId, "supplier_operator_address", supplierOperatorAddress).
		Add(1)
}

// CaptureDelayedRelayRequestRateLimitingCheck records a rate limiting event during delayed validation.
// This metric is incremented when relay requests are rate limited during the delayed
// validation process, typically when the application exceeds its allocated stake
// during delayed validation checks.
func CaptureDelayedRelayRequestRateLimitingCheck(serviceId string, supplierOperatorAddress string) {
	DelayedRelayRequestRateLimitingCheckTotal.
		With("service_id", serviceId, "supplier_operator_address", supplierOperatorAddress).
		Add(1)
}
