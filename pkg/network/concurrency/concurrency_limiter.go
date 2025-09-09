// Concurrency Limiter for Resource Management
// ===========================================
//
// This concurrency limiter implements a semaphore pattern to bound the number
// of concurrent HTTP operations, preventing resource exhaustion under high load.
//
// When processing thousands of simultaneous HTTP requests, unlimited concurrency
// can overwhelm system resources (memory, file descriptors, network connections).
//
// Resource Protection Mechanisms:
//   - Semaphore-based admission control using buffered channels
//   - Context-aware blocking with cancellation support
//   - Real-time tracking of active request counts
//   - Graceful degradation when limits are exceeded
//
// Operational Characteristics:
//   - Blocks new requests when limit is reached
//   - Respects context cancellation for timeout handling
//   - Integrates with metrics for observability
//   - Thread-safe for concurrent access
//
// The limiter prevents cascading failures by ensuring system resources remain
// available even during traffic spikes or slow downstream services.
package concurrency

import (
	"context"
	"sync"
	"time"
)

// TODO_IMPROVE: Make this configurable via settings
const (
	defaultMaxConcurrentRequests = 1_000_000
)

// ConcurrencyLimiter bounds concurrent operations via semaphore pattern.
// Prevents resource exhaustion and tracks active request counts.
type ConcurrencyLimiter struct {
	semaphore      chan struct{}
	maxConcurrent  int
	activeRequests int64
	mu             sync.RWMutex
}

// NewConcurrencyLimiter creates a limiter that bounds concurrent operations.
func NewConcurrencyLimiter(maxConcurrent int) *ConcurrencyLimiter {
	if maxConcurrent <= 0 {
		maxConcurrent = defaultMaxConcurrentRequests // Default reasonable limit
	}

	return &ConcurrencyLimiter{
		semaphore:     make(chan struct{}, maxConcurrent),
		maxConcurrent: maxConcurrent,
	}
}

// TODO_TECHDEBT(@adshmh): Track active relays for observability
//
// Acquire blocks until a slot is available or context is canceled.
// Returns true if acquired, false if context was canceled.
func (cl *ConcurrencyLimiter) Acquire(ctx context.Context) bool {
	select {
	case cl.semaphore <- struct{}{}:
		cl.mu.Lock()
		cl.activeRequests++
		cl.mu.Unlock()
		return true
	case <-ctx.Done():
		return false
	}
}

// tryAcquireWithTimeout attempts to acquire a slot with timeout.
func (cl *ConcurrencyLimiter) tryAcquireWithTimeout(timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return cl.Acquire(ctx)
}

// Release returns a slot to the pool.
func (cl *ConcurrencyLimiter) Release() {
	select {
	case <-cl.semaphore:
		cl.mu.Lock()
		cl.activeRequests--
		cl.mu.Unlock()
	default:
		// TODO_TECHDEBT: Log acquire/release mismatch for debugging
	}
}

// getActiveRequests returns the current number of active requests.
func (cl *ConcurrencyLimiter) getActiveRequests() int64 {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	return cl.activeRequests
}
