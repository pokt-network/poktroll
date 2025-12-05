package observability

import (
	"time"
)

// Timer is a helper for measuring operation durations.
type Timer struct {
	startTime time.Time
}

// NewTimer creates a new timer starting from now.
func NewTimer() *Timer {
	return &Timer{startTime: time.Now()}
}

// ObserveOperation records the duration to the OperationDurationSeconds histogram.
func (t *Timer) ObserveOperation(component, operation, status string) {
	OperationDurationSeconds.WithLabelValues(component, operation, status).Observe(time.Since(t.startTime).Seconds())
}

// ObserveRedisOperation records the duration to the RedisOperationDurationSeconds histogram.
func (t *Timer) ObserveRedisOperation(operation, status string) {
	RedisOperationDurationSeconds.WithLabelValues(operation, status).Observe(time.Since(t.startTime).Seconds())
	RedisOperationsTotal.WithLabelValues(operation, status).Inc()
}

// ObserveOnchainQuery records the duration to the OnchainQueryDurationSeconds histogram.
func (t *Timer) ObserveOnchainQuery(queryType, status string) {
	OnchainQueryDurationSeconds.WithLabelValues(queryType, status).Observe(time.Since(t.startTime).Seconds())
	OnchainQueriesTotal.WithLabelValues(queryType, status).Inc()
}

// ObserveTxSubmission records the duration to the TxSubmissionDurationSeconds histogram.
func (t *Timer) ObserveTxSubmission(txType, status string) {
	TxSubmissionDurationSeconds.WithLabelValues(txType, status).Observe(time.Since(t.startTime).Seconds())
	TxSubmissionsTotal.WithLabelValues(txType, status).Inc()
}

// ObserveSigning records the duration to the SigningDurationSeconds histogram.
func (t *Timer) ObserveSigning(operation string) {
	SigningDurationSeconds.WithLabelValues(operation).Observe(time.Since(t.startTime).Seconds())
}

// Duration returns the elapsed time since the timer was started.
func (t *Timer) Duration() time.Duration {
	return time.Since(t.startTime)
}

// RecordCacheHit increments the cache hit counter.
func RecordCacheHit(cacheName string) {
	CacheOperationsTotal.WithLabelValues(cacheName, "hit").Inc()
}

// RecordCacheMiss increments the cache miss counter.
func RecordCacheMiss(cacheName string) {
	CacheOperationsTotal.WithLabelValues(cacheName, "miss").Inc()
}

// RecordError increments the error counter.
func RecordError(component, errorType string) {
	ErrorsTotal.WithLabelValues(component, errorType).Inc()
}

// SetQueueMetrics updates queue depth and capacity.
func SetQueueMetrics(queueName string, depth, capacity int) {
	QueueDepth.WithLabelValues(queueName).Set(float64(depth))
	QueueCapacity.WithLabelValues(queueName).Set(float64(capacity))
}

// SetMemoryUsage updates memory usage for a component.
func SetMemoryUsage(component string, bytes int64) {
	MemoryUsageBytes.WithLabelValues(component).Set(float64(bytes))
}

// SetGoroutineCount updates the goroutine count for a component.
func SetGoroutineCount(component string, count int) {
	GoroutineCount.WithLabelValues(component).Set(float64(count))
}

// RecordStartupDuration records how long a component took to start.
func RecordStartupDuration(component string, duration time.Duration) {
	StartupDurationSeconds.WithLabelValues(component).Set(duration.Seconds())
}

// SetProcessInfo sets static process information.
func SetProcessInfo(version, component string) {
	ProcessInfo.WithLabelValues(version, component).Set(1)
}
