package proxy

import (
	stdprometheus "github.com/prometheus/client_golang/prometheus"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
)

var (
	// relaysTotal is a counter metric representing the total number of requests processed by the relay miner.
	// As a Counter, it only increments and is used to track the occurrence of an event (in this case, proxy requests).
	// The metric is labeled by 'proxy_name' and 'service_id', allowing for differentiation of request counts
	// across various proxies and services. This is crucial for monitoring and understanding the load and traffic
	// patterns on the relay miner, providing insights into usage and potential bottlenecks.
	//
	// Example of usage:
	// - Monitoring the total request load over time.
	// - Comparing request counts across different services or proxy instances.
	relaysTotal metrics.Counter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: "relayminer",
		Name:      "requests_total",
	}, []string{"proxy_name", "service_id"})

	// relaysDurationSeconds is a histogram metric used to observe the distribution of request durations in the relay miner.
	// It measures the time taken for requests to be processed, in seconds. This metric is vital for performance monitoring,
	// as it helps in understanding the response time characteristics of the relay miner under different load conditions.
	// The histogram is labeled by 'proxy_name' and 'service_id', enabling detailed analysis per proxy instance and service.
	// Analyzing the distribution of request durations can help in identifying performance bottlenecks and areas for optimization.
	//
	// The buckets for this histogram are configured as follows:
	// - 0.1 seconds: Captures very fast responses.
	// - 0.5 seconds: Suitable for moderately fast responses.
	// - 1 second: Standard response time.
	// - 2 seconds: Slower, but acceptable response time.
	// - 5 seconds: Captures relatively long response times.
	// - 15 seconds: Captures the upper limit of expected response times.
	// This range of buckets was chosen to balance granularity with the need to avoid high cardinality, capturing a broad spectrum of response times while keeping the number of unique time series manageable.
	//
	// Example of usage:
	// - Determining the typical request response times.
	// - Identifying long-tail latency issues or outliers in request processing.
	// - Comparing performance across different services or deployment environments.
	//
	// Note: It is recommended to place the bucket sizes in a configuration file for easier,
	// dynamic adjustments without source code modification, which is particularly useful
	// for adapting to different data patterns or deployment environments.
	relaysDurationSeconds metrics.Histogram = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Subsystem: "relayminer",
		Name:      "request_duration_seconds",
		Buckets:   []float64{0.1, 0.5, 1, 2, 5, 15},
	}, []string{"proxy_name", "service_id"})

	// TODO(@okdas): add `response_size_bytes`. Skipping for now to avoid creating of a new HTTP server writer - let's
	// reevaluate after we have some historical load testing data.
	// // responseSizeBytes defines a histogram metric to observe the size distribution of proxy response sizes.
	// // Each bucket boundary in the array represents the upper inclusive limit for that bucket.
	// // This histogram counts the number of responses, measuring each response's size in bytes. The bucket ranges are:
	// // - 100 bytes: Counts all responses up to 100 bytes.
	// // - 500 bytes: Counts responses more than 100 bytes but up to 500 bytes.
	// // - 1,000 bytes (1 KB): Counts responses more than 500 bytes but up to 1 KB.
	// // - 5,000 bytes (5 KB): Counts responses more than 1 KB but up to 5 KB.
	// // - 10,000 bytes (10 KB): Counts responses more than 5 KB but up to 10 KB.
	// // - 50,000 bytes (50 KB): Counts responses more than 10 KB but up to 50 KB.
	// // Responses larger than 50 KB are counted in the last bucket.
	// // These initial bucket sizes should be revisited and adjusted based on actual observed data
	// // to ensure accurate representation of the distribution of response sizes.
	// //
	// // Note: It is recommended to place the bucket sizes in a configuration file for easier,
	// // dynamic adjustments without source code modification, which is particularly useful
	// // for adapting to different data patterns or deployment environments.
	// responseSizeBytes metrics.Histogram = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
	// 	Subsystem: "relayminer",
	// 	Name:      "response_size_bytes",
	// 	Buckets:   []float64{100, 500, 1000, 5000, 10000, 50000},
	// }, []string{"proxy_name", "service_id"})
)
