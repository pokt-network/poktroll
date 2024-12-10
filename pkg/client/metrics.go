package client

import (
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	clientSubsystem = "client"

	allQueriesTotal    = "all_queries_total"
	paramsQueriesTotal = "params_queries_total"
)

var (
	// TODO_IN_THIS_COMMIT: godoc...
	AllQueriesTotalCounter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: clientSubsystem,
		Name:      allQueriesTotal,
		Help:      "Total number of all query messages, of any type, sent by the client.",
		// TODO_IN_THIS_COMMIT: extract labels to constants.
		//}, []string{"client_type", "method", "msg_type", "claim_proof_lifecycle_stage"})
	}, []string{"client_type", "method", "msg_type"})

	// TODO_IN_THIS_COMMIT: godoc...
	ParamsQueriesTotalCounter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: clientSubsystem,
		Name:      paramsQueriesTotal,
		Help:      "Total number of QueryParamsRequest messages sent by the client.",
	}, []string{"client_type", "method", "claim_proof_lifecycle_stage"})

	// TODO_IN_THIS_COMMIT: godoc & use...
	AllWebsocketEventsTotalCounter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Subsystem: clientSubsystem,
		Name:      "all_websocket_events_total",
		Help:      "Total number of websocket events received by the client.",
	}, []string{"client_type", "event_type"})
)
