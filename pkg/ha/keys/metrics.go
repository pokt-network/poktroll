package keys

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	metricsNamespace = "ha"
	metricsSubsystem = "keys"
)

var (
	supplierKeysActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "supplier_keys_active",
			Help:      "Number of active supplier signing keys",
		},
	)

	keyReloadsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "reloads_total",
			Help:      "Total number of key reloads",
		},
	)

	keyChangesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "changes_total",
			Help:      "Total number of key changes",
		},
		[]string{"type"}, // type: added, removed
	)

	keyLoadErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "load_errors_total",
			Help:      "Total number of key load errors",
		},
		[]string{"provider"},
	)
)
