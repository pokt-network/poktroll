package appgateserver

import (
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	appGateServerProcess = "appgateserver"

	relaysTotalMetric        = "relay_requests_total"
	relaysSuccessTotalMetric = "relay_requests_success_total"
	relaysErrorsTotalMetric  = "relay_requests_errors_total"
)

var (
	// relaysTotal is a Counter metric for the total relays processed by the AppGate server.
	// Crucial for understanding server workload and traffic, it increments monotonically.
	// Labeled by 'service_id' and 'rpc_type', it facilitates nuanced analysis of relays
	// across various services and RPC types.
	//
	// Usage:
	// - Monitor aggregate load and relay rates.
	// - Compare relay volumes by service and RPC type.
	relaysTotal = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: appGateServerProcess,
		Name:      relaysTotalMetric,
	}, []string{"service_id", "rpc_type"})

	// relaysErrorsTotal is a Counter metric tracking errors on the AppGate server.
	// Incrementing with each error, it's vital for server health and stability assessment.
	// With 'service_id' and 'rpc_type' labels, it allows precise error rate analysis and troubleshooting
	// across services and RPC types.
	//
	// Usage:
	// - Monitor health and error rates by service and RPC type.
	// - Identify and address high-error areas.
	relaysErrorsTotal = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: appGateServerProcess,
		Name:      relaysErrorsTotalMetric,
	}, []string{"service_id", "rpc_type"})

	// relaysSuccessTotal is a Counter metric tracking successful relays on the AppGate server.
	// It's essential for monitoring server reliability and performance.
	// Labeled by 'service_id' and 'rpc_type', it enables detailed analysis of successful requests.
	relaysSuccessTotal = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: appGateServerProcess,
		Name:      relaysSuccessTotalMetric,
	}, []string{"service_id", "rpc_type"})
)
