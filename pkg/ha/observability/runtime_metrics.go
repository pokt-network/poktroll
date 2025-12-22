package observability

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var (
	// Runtime metrics
	runtimeGoroutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "goroutines",
			Help:      "Number of goroutines",
		},
	)

	runtimeThreads = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "threads",
			Help:      "Number of OS threads",
		},
	)

	runtimeHeapAlloc = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "heap_alloc_bytes",
			Help:      "Bytes of allocated heap objects",
		},
	)

	runtimeHeapSys = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "heap_sys_bytes",
			Help:      "Bytes of heap memory obtained from the OS",
		},
	)

	runtimeHeapIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "heap_idle_bytes",
			Help:      "Bytes in idle (unused) spans",
		},
	)

	runtimeHeapInuse = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "heap_inuse_bytes",
			Help:      "Bytes in in-use spans",
		},
	)

	runtimeHeapReleased = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "heap_released_bytes",
			Help:      "Bytes of physical memory returned to the OS",
		},
	)

	runtimeHeapObjects = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "heap_objects",
			Help:      "Number of allocated heap objects",
		},
	)

	runtimeStackInuse = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "stack_inuse_bytes",
			Help:      "Bytes in stack spans",
		},
	)

	runtimeStackSys = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "stack_sys_bytes",
			Help:      "Bytes of stack memory obtained from the OS",
		},
	)

	runtimeMallocs = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "mallocs_total",
			Help:      "Cumulative count of heap objects allocated",
		},
	)

	runtimeFrees = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "frees_total",
			Help:      "Cumulative count of heap objects freed",
		},
	)

	runtimeGCPauseTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "gc_pause_total_nanoseconds",
			Help:      "Cumulative nanoseconds in GC stop-the-world pauses",
		},
	)

	runtimeNumGC = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "gc_completed_total",
			Help:      "Number of completed GC cycles",
		},
	)

	runtimeNumForcedGC = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "gc_forced_total",
			Help:      "Number of GC cycles forced by the application",
		},
	)

	runtimeLastGC = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "last_gc_timestamp_seconds",
			Help:      "Timestamp of last GC",
		},
	)

	runtimeNextGC = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "next_gc_heap_size_bytes",
			Help:      "Target heap size of the next GC cycle",
		},
	)

	runtimeGCCPUFraction = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "runtime",
			Name:      "gc_cpu_fraction",
			Help:      "Fraction of CPU time used by GC",
		},
	)
)

// RuntimeMetricsCollectorConfig configures the runtime metrics collector.
type RuntimeMetricsCollectorConfig struct {
	// CollectionInterval is how often to collect runtime metrics.
	CollectionInterval time.Duration
}

// DefaultRuntimeMetricsCollectorConfig returns sensible defaults.
func DefaultRuntimeMetricsCollectorConfig() RuntimeMetricsCollectorConfig {
	return RuntimeMetricsCollectorConfig{
		CollectionInterval: 10 * time.Second,
	}
}

// RuntimeMetricsCollector periodically collects Go runtime metrics.
type RuntimeMetricsCollector struct {
	logger polylog.Logger
	config RuntimeMetricsCollectorConfig

	// Previous values for delta calculations
	lastMallocs      uint64
	lastFrees        uint64
	lastGCPauseTotal uint64
	lastNumGC        uint32
	lastNumForcedGC  uint32

	// Lifecycle
	ctx      context.Context
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// NewRuntimeMetricsCollector creates a new runtime metrics collector.
func NewRuntimeMetricsCollector(
	logger polylog.Logger,
	config RuntimeMetricsCollectorConfig,
) *RuntimeMetricsCollector {
	if config.CollectionInterval == 0 {
		config.CollectionInterval = 10 * time.Second
	}

	return &RuntimeMetricsCollector{
		logger: logging.ForComponent(logger, logging.ComponentRuntimeMetrics),
		config: config,
	}
}

// Start begins collecting runtime metrics.
func (c *RuntimeMetricsCollector) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	c.ctx, c.cancelFn = context.WithCancel(ctx)
	c.running = true

	// Collect initial values
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	c.lastMallocs = memStats.Mallocs
	c.lastFrees = memStats.Frees
	c.lastGCPauseTotal = memStats.PauseTotalNs
	c.lastNumGC = memStats.NumGC
	c.lastNumForcedGC = memStats.NumForcedGC

	c.wg.Add(1)
	go c.collectLoop()

	c.logger.Info().
		Dur("collection_interval", c.config.CollectionInterval).
		Msg("runtime metrics collector started")

	return nil
}

// Stop stops collecting runtime metrics.
func (c *RuntimeMetricsCollector) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	c.cancelFn()
	c.mu.Unlock()

	c.wg.Wait()
	c.logger.Info().Msg("runtime metrics collector stopped")
}

// collectLoop periodically collects runtime metrics.
func (c *RuntimeMetricsCollector) collectLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collect()
		}
	}
}

// collect reads runtime metrics and updates Prometheus gauges.
func (c *RuntimeMetricsCollector) collect() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Goroutines and threads
	runtimeGoroutines.Set(float64(runtime.NumGoroutine()))
	runtimeThreads.Set(float64(runtime.GOMAXPROCS(0)))

	// Heap metrics
	runtimeHeapAlloc.Set(float64(memStats.HeapAlloc))
	runtimeHeapSys.Set(float64(memStats.HeapSys))
	runtimeHeapIdle.Set(float64(memStats.HeapIdle))
	runtimeHeapInuse.Set(float64(memStats.HeapInuse))
	runtimeHeapReleased.Set(float64(memStats.HeapReleased))
	runtimeHeapObjects.Set(float64(memStats.HeapObjects))

	// Stack metrics
	runtimeStackInuse.Set(float64(memStats.StackInuse))
	runtimeStackSys.Set(float64(memStats.StackSys))

	// Allocation deltas (as counters)
	if memStats.Mallocs > c.lastMallocs {
		runtimeMallocs.Add(float64(memStats.Mallocs - c.lastMallocs))
		c.lastMallocs = memStats.Mallocs
	}
	if memStats.Frees > c.lastFrees {
		runtimeFrees.Add(float64(memStats.Frees - c.lastFrees))
		c.lastFrees = memStats.Frees
	}

	// GC metrics
	if memStats.PauseTotalNs > c.lastGCPauseTotal {
		runtimeGCPauseTotal.Add(float64(memStats.PauseTotalNs - c.lastGCPauseTotal))
		c.lastGCPauseTotal = memStats.PauseTotalNs
	}
	if memStats.NumGC > c.lastNumGC {
		runtimeNumGC.Add(float64(memStats.NumGC - c.lastNumGC))
		c.lastNumGC = memStats.NumGC
	}
	if memStats.NumForcedGC > c.lastNumForcedGC {
		runtimeNumForcedGC.Add(float64(memStats.NumForcedGC - c.lastNumForcedGC))
		c.lastNumForcedGC = memStats.NumForcedGC
	}

	runtimeLastGC.Set(float64(memStats.LastGC) / 1e9)
	runtimeNextGC.Set(float64(memStats.NextGC))
	runtimeGCCPUFraction.Set(memStats.GCCPUFraction)
}

// CollectNow triggers an immediate collection of runtime metrics.
func (c *RuntimeMetricsCollector) CollectNow() {
	c.collect()
}
