package retry

import (
	"context"
	"math"
	"time"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var WithDefaultExponentialDelay = WithExponentialBackoffFn(5, 500, 30000)

type RetryFunc func() chan error
type RetryStrategyFunc func(int) bool

// OnError continuously invokes the provided work function (workFn) until either the context (ctx)
// is canceled or the error channel returned by workFn is closed. If workFn encounters an error,
// OnError will retry invoking workFn based on the provided retry parameters.
//
// Parameters:
//   - ctx: the context to monitor for cancellation. If canceled, OnError will exit without error.
//   - retryLimit: the maximum number of retries for workFn upon encountering an error.
//   - retryDelay: the duration to wait before retrying workFn after an error.
//   - retryResetCount: Specifies the duration of continuous error-free operation required
//     before the retry count is reset. If the work function operates without
//     errors for this duration, any subsequent error will restart the retry
//     count from the beginning.
//   - workName: a name or descriptor for the work function, used for logging purposes.
//   - workFn: a function that performs some work and returns an error channel.
//     This channel emits errors encountered during the work.
//
// Returns:
// - If the context is canceled, the function returns nil.
// - If the error channel is closed, a warning is logged, and the function returns nil.
// - If the retry limit is reached, the function returns the error from the channel.
//
// Note: After each error, a delay specified by retryDelay is introduced before retrying workFn.func OnError(
func OnError(
	ctx context.Context,
	retryLimit int,
	retryDelay time.Duration,
	retryResetTimeout time.Duration,
	workName string,
	workFn RetryFunc,
) error {
	logger := polylog.Ctx(ctx)

	var retryCount int
	errCh := workFn()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(retryResetTimeout):
			retryCount = 0
		case err, ok := <-errCh:
			// Exit the retry loop if the error channel is closed.
			if !ok {
				logger.Warn().
					Str("work_name", workName).
					Msg("error channel closed, will no longer retry on error")
				return nil
			}

			// Return error if retry limit reached
			// A negative retryLimit allows limitless retries
			if retryLimit >= 0 && retryCount >= retryLimit {
				return err
			}

			// Wait retryDelay before retrying.
			time.Sleep(retryDelay)

			// Increment retryCount and retry workFn.
			retryCount++
			errCh = workFn()
			logger.Error().
				Str("work_name", workName).
				Err(err).
				Msgf("on retry: %d", retryCount)
		}
	}
}

// Call executes a function repeatedly until it succeeds or the retry strategy
// indicates that no more retries should be attempted.
//
// If no retry strategy is provided, it defaults to an exponential backoff strategy
// with 5 max retries, 1000ms initial delay, and 16000ms max delay.
//
// Returns the result from the work function and any error that occurred if retries are exhausted.
func Call[T any](
	work func() (T, error),
	retryStrategy ...RetryStrategyFunc,
) (T, error) {
	var result T
	var err error
	if retryStrategy == nil {
		retryStrategy = []RetryStrategyFunc{WithDefaultExponentialDelay}
	}

	for retryCount := 0; ; retryCount++ {
		result, err = work()
		if err == nil {
			return result, nil
		}
		if !retryStrategy[0](retryCount) {
			return result, err
		}
	}
}

// paramsGetter defines an interface for types that can retrieve parameters of type T.
// Implementations must provide a GetParams method that accepts a context and returns
// the retrieved parameters along with any error that occurred during retrieval.
type paramsGetter[T any] interface {
	GetParams(ctx context.Context) (T, error)
}

// GetParams retrieves parameters from a paramsGetter with automatic retry handling.
//
// This function wraps the GetParams method of the provided paramsGetter with retry
// logic using the Call function. If the initial attempt to get parameters fails,
// subsequent attempts will be made according to the provided retry strategy.
func GetParams[T any](
	ctx context.Context,
	paramsGetter paramsGetter[T],
	retryStrategy ...RetryStrategyFunc,
) (T, error) {
	// Wrap the GetParams call in a function that matches the signature expected by Call
	// This adapts the paramsGetter.GetParams method to be used with our retry mechanism
	return Call(
		func() (T, error) { return paramsGetter.GetParams(ctx) },
		retryStrategy...,
	)
}

// WithExponentialBackoffFn creates a retry strategy with exponential backoff.
func WithExponentialBackoffFn(
	maxRetryCount int,
	initialDelayMs int,
	maxDelayMs int,
) func(retryCount int) bool {
	return func(retryCount int) bool {
		if retryCount >= maxRetryCount {
			return false
		}

		backoffDelay := initialDelayMs * int(math.Pow(2, float64(retryCount)))
		delay := min(backoffDelay, maxDelayMs)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		return true
	}
}

// UntilNextBlock creates a retry strategy function that retries until a new block
// is observed or falls back to the provided retry strategy.
//
// The function subscribes to a block observable and creates a retry strategy that
// will either retry based on the provided strategy or stop retrying when a new block
// is observed, whichever happens first.
func UntilNextBlock(
	ctx context.Context,
	blockObservable observable.Observable[client.Block],
	retryStrategy ...RetryStrategyFunc,
) func(retryCount int) bool {
	// Create a context that can be canceled when a block is observed
	ctx, cancel := context.WithCancel(ctx)
	// Subscribe to the block observable to get notified of new blocks
	heightReachedCh := blockObservable.Subscribe(ctx).Ch()

	// Use default retry strategy if none is provided
	if retryStrategy == nil {
		retryStrategy = []RetryStrategyFunc{WithDefaultExponentialDelay}
	}

	return func(retryCount int) bool {
		// Channel to signal when the retry strategy has completed its delay
		retryCh := make(chan struct{})
		defer close(retryCh)

		// Execute the retry strategy in a goroutine to avoid blocking
		go func() {
			retryStrategy[0](retryCount)
			// Signal that the retry strategy has completed
			retryCh <- struct{}{}
		}()

		// Wait for one of three conditions: context done, new block observed, or retry strategy completed
		for {
			select {
			case <-ctx.Done():
				// The context was canceled, don't retry
				return false
			case <-heightReachedCh:
				// A new block was observed, cancel the context and don't retry
				cancel()
				return false
			case <-retryCh:
				// The retry strategy completed, signal to retry the operation
				return true
			}
		}
	}
}
