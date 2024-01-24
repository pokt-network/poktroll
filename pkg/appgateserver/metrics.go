package appgateserver

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	// relaysTotal is a counter metric that represents the total number of requests processed by the AppGate server.
	// It is a Counter type metric, which means it is used to accumulate counts of events (in this case, the total number of requests).
	// This metric is crucial for understanding the overall workload and traffic handled by the server.
	// It is labeled by 'service_id', which allows for distinguishing and aggregating request counts across different services managed by the AppGate server.
	//
	// Example of usage:
	// - Monitoring the aggregate load and request rate over time.
	// - Comparing request volumes across different services managed by the AppGate server.
	relaysTotal metrics.Counter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: "appgateserver",
		Name:      "requests_total",
	}, []string{"service_id", "request_type"})

	// relaysErrorsTotal is a counter metric that tracks the total number of errors encountered by the AppGate server.
	// This metric increments each time an error occurs, providing insight into the health and stability of the server.
	// Tracking error rates is essential for maintaining the reliability of the server and for identifying issues that require attention.
	// The metric is labeled by 'service_id', which allows for tracking and analyzing error rates per service,
	// enabling targeted troubleshooting and performance optimization.
	//
	// Example of usage:
	// - Monitoring the overall health and error rates of the server.
	// - Identifying services with higher error rates for targeted debugging and improvement.
	relaysErrorsTotal metrics.Counter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: "appgateserver",
		Name:      "errors_total",
	}, []string{"service_id"})
)
