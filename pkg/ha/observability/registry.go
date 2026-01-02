package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Component-specific registries to ensure clean metric separation.
// The miner should only expose miner metrics, and the relayer should only expose relayer metrics.
var (
	// MinerRegistry is the Prometheus registry for miner metrics.
	MinerRegistry = prometheus.NewRegistry()

	// RelayerRegistry is the Prometheus registry for relayer metrics.
	RelayerRegistry = prometheus.NewRegistry()

	// SharedRegistry is for metrics shared between components (cache, keys, etc.)
	SharedRegistry = prometheus.NewRegistry()

	// MinerFactory creates metrics registered to the miner registry.
	MinerFactory = promauto.With(MinerRegistry)

	// RelayerFactory creates metrics registered to the relayer registry.
	RelayerFactory = promauto.With(RelayerRegistry)

	// SharedFactory creates metrics registered to the shared registry.
	SharedFactory = promauto.With(SharedRegistry)
)

func init() {
	// Register standard Go metrics collectors to both registries
	MinerRegistry.MustRegister(prometheus.NewGoCollector())
	MinerRegistry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	RelayerRegistry.MustRegister(prometheus.NewGoCollector())
	RelayerRegistry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
}
