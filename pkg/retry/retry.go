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

type paramsGetter[T any] interface {
	GetParams(ctx context.Context) (T, error)
}

func GetParams[T any](
	ctx context.Context,
	paramsGetter paramsGetter[T],
	retryStrategy ...RetryStrategyFunc,
) (T, error) {
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
		if retryCount > maxRetryCount {
			return false
		}

		backoffDelay := initialDelayMs * int(math.Pow(2, float64(retryCount)))
		delay := min(backoffDelay, maxDelayMs)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		return true
	}
}

// WithBlockHeightLimitFn creates a retry strategy that stops retrying once a specified
// block height is reached.
// It delegates the retry delay logic to the provided retryStrategy function.
func WithBlockHeightLimitFn(
	ctx context.Context,
	blockObservable observable.Observable[client.Block],
	maxBlockHeight int,
	retryStrategy ...RetryStrategyFunc,
) func(retryCount int) bool {
	ctx, cancel := context.WithCancel(ctx)
	blockCh := blockObservable.Subscribe(ctx).Ch()

	// If no retry strategy is provided, use the default exponential backoff strategy.
	// This is a fallback in case the caller does not provide a strategy.
	if retryStrategy == nil {
		retryStrategy = []RetryStrategyFunc{WithDefaultExponentialDelay}
	}

	return func(retryCount int) bool {
		retryCh := make(chan struct{})
		defer close(retryCh)

		go func() {
			// Wait for the retry strategy to complete.
			retryStrategy[0](retryCount)
			retryCh <- struct{}{}
		}()

		for {
			// Wait for the context to be canceled or the block height limit to be reached.
			select {
			case <-ctx.Done():
				return false
			case block := <-blockCh:
				if block.Height() > int64(maxBlockHeight) {
					cancel()
					return false
				}
			case <-retryCh:
				// Delegate the retry delays to the underlying strategy.
				// Do not rely on the retry strategy to stop retrying.
				return true
			}
		}
	}
}

func UntilNextBlock(
	ctx context.Context,
	blockObservable observable.Observable[client.Block],
	retryStrategy ...RetryStrategyFunc,
) func(retryCount int) bool {
	ctx, cancel := context.WithCancel(ctx)
	heightReachedCh := blockObservable.Subscribe(ctx).Ch()

	if retryStrategy == nil {
		retryStrategy = []RetryStrategyFunc{WithDefaultExponentialDelay}
	}

	return func(retryCount int) bool {
		retryCh := make(chan struct{})
		defer close(retryCh)

		go func() {
			retryStrategy[0](retryCount)
			retryCh <- struct{}{}
		}()

		for {
			select {
			case <-ctx.Done():
				return false
			case <-heightReachedCh:
				cancel()
				return false
			case <-retryCh:
				return true
			}
		}
	}
}
