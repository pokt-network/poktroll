package appgateserver

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	appGateServerSubsystem = "appgateserver"

	requestsTotal        = "requests_total"
	requestsSuccessTotal = "requests_success_total"
	requestsErrorsTotal  = "requests_errors_total"
)

var (
	// relaysTotal is a Counter metric for the total requests processed by the AppGate server.
	// Crucial for understanding server workload and traffic, it increments monotonically.
	// Labeled by 'service_id' and 'request_type', it facilitates nuanced analysis of requests
	// across various services and request types.
	//
	// Usage:
	// - Monitor aggregate load and request rates.
	// - Compare request volumes by service and request type.
	relaysTotal metrics.Counter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: appGateServerSubsystem,
		Name:      requestsTotal,
	}, []string{"service_id", "request_type"})

	// relaysErrorsTotal is a Counter metric tracking errors on the AppGate server.
	// Incrementing with each error, it's vital for server health and stability assessment.
	// With 'service_id' and 'request_type' labels, it allows precise error rate analysis and troubleshooting
	// across services and request types.
	//
	// Usage:
	// - Monitor health and error rates by service and request type.
	// - Identify and address high-error areas.
	relaysErrorsTotal metrics.Counter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: appGateServerSubsystem,
		Name:      requestsErrorsTotal,
	}, []string{"service_id", "request_type"})

	// relaysSuccessTotal is a Counter metric tracking successful requests on the AppGate server.
	// Incrementing with each successful request, it's essential for monitoring server
	// reliability and performance.
	// Labeled by 'service_id' and 'request_type', it enables detailed analysis
	// of successful requests.
	relaysSuccessTotal metrics.Counter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: appGateServerSubsystem,
		Name:      requestsSuccessTotal,
	}, []string{"service_id", "request_type"})
)
