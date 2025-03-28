package retry

import (
	"context"
	"math"
	"time"

	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

const RetryStrategyCtxKey = "retry_strategy"

var DefaultExponentialDelay = WithExponentialBackoffFn(25, 500, 30000)

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
	var (
		result T
		err    error
	)
	if retryStrategy == nil {
		retryStrategy = []RetryStrategyFunc{DefaultExponentialDelay}
	}

	for retryCount := 0; ; retryCount++ {
		result, err = work()

		// Stop retrying and return the result if no error occurred
		if err == nil {
			return result, err
		}

		// Stop retrying and return the result if the error is non-retryable
		if ErrNonRetryable.Is(err) {
			return result, err
		}

		// Stop retrying and return if the error is a gRPC error
		if _, isGRPCError := status.FromError(err); isGRPCError {
			return result, err
		}

		if !retryStrategy[0](retryCount) {
			return result, err
		}
	}
}

// GetStrategy retrieves the retry strategy from the context.
// If no strategy is found, it defaults to the default exponential delay strategy.
// This function is useful for setting a custom retry strategy in the context
// and retrieving it later in the code execution.
func GetStrategy(ctx context.Context) RetryStrategyFunc {
	strategy, ok := ctx.Value(RetryStrategyCtxKey).(RetryStrategyFunc)
	if !ok {
		return DefaultExponentialDelay
	}
	return strategy
}

// WithExponentialBackoffFn creates a retry strategy with exponential backoff.
func WithExponentialBackoffFn(
	maxRetryCount int,
	initialDelayMs int,
	maxDelayMs int,
) func(retryCount int) bool {
	return func(retryCount int) bool {
		// Higher level strategies (e.g. retry until height reached) could use this
		// one while ignoring the maxRetryCount.
		// Capping the delay to maxDelayMs to prevent excessive wait times.
		if retryCount >= maxRetryCount {
			time.Sleep(time.Duration(maxDelayMs) * time.Millisecond)
			return false
		}

		backoffDelay := initialDelayMs * int(math.Pow(2, float64(retryCount)))
		delay := min(backoffDelay, maxDelayMs)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		return true
	}
}
