package sync2

import (
	"context"
	"sync"
	"sync/atomic"
)

type CircuitBreaker struct {
	// TODO: add a noCopy field?

	goGroup sync.WaitGroup
	// goCount is the number of goroutines currently running.
	goCount *atomic.Int64
	// limit is the maximum number of goroutines allowed to run.
	limit  uint
	close  sync.Once
	closed chan struct{}
}

func NewCircuitBreaker(limit uint) *CircuitBreaker {
	return &CircuitBreaker{
		goGroup: sync.WaitGroup{},
		goCount: new(atomic.Int64),
		// NB: convert limit to int64 so that it can be decremented atomically.
		limit:  limit,
		closed: make(chan struct{}),
	}
}

func (cb *CircuitBreaker) Go(ctx context.Context, fn func()) bool {
	if ctx.Err() != nil {
		return false
	}

	select {
	case <-cb.closed:
		return false
	case <-ctx.Done():
		return false
	default:
	}

	if cb.goCount.Add(1) > int64(cb.limit) {
		return false
	}

	cb.goGroup.Add(1)

	go func() {
		defer func() {
			cb.goCount.Add(-1)
			cb.goGroup.Done()
		}()
		fn()
	}()

	return true
}

func (cb *CircuitBreaker) Close() {
	cb.close.Do(func() {
		close(cb.closed)
		// Wait for all running goroutines to finish.
		cb.goGroup.Wait()
	})
}
