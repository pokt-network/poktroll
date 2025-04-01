package retry

import (
	"context"
	"math"
	"slices"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

const (
	RetryStrategyCtxKey = "retry_strategy"

	// Exponential backoff factor used in the retry strategy formula:
	// initial * exponentialBackOffFactor^retryCount
	exponentialBackOffFactor = 2.0

	// TODO_IMPROVE: Make these configurable via flags, configs or env vars.
	DefaultMaxRetryCount  = 25
	DefaultInitialDelayMs = 500
	DefaultMaxDelayMs     = 30000
)

var (
	// Prepare the default exponential backoff strategy
	DefaultExponentialDelay = WithExponentialBackoffFn(DefaultMaxDelayMs, DefaultInitialDelayMs, DefaultMaxDelayMs)

	// transientGRPCErrorCodes is a list of gRPC error codes that are considered transient.
	// These errors are retried by default.
	transientGRPCErrorCodes = []codes.Code{

		// This is most likely a transient condition and may be corrected by retrying with a backoff.
		// Occurs during server shutdowns or network connectivity issues.
		codes.Unavailable,

		// Indicates temporary resource limitations (quotas, memory, server overload).
		// Resources may become available after some time.
		codes.ResourceExhausted,

		// Often indicates temporary slowness or timeouts.
		// May succeed on retry when system load decreases.
		codes.DeadlineExceeded,

		// Typically caused by concurrency issues like transaction conflicts
		codes.Aborted,

		// May be transient, but cause is unclear
		codes.Unknown,

		// Server-side errors that might occasionally resolve with retry
		codes.Internal,
	}
)

type RetryFunc func() chan error
type RetryStrategyFunc func(context.Context, int) bool

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

// Call executes a function repeatedly according to the retry strategy until either:
// - It succeeds
// - It returns an `ErrNonRetryable` (indicating no more retries should be attempted)
//
// If no retry strategy is provided, it defaults to an exponential backoff strategy defined at the top of this file.
//
// Returns:
// - The result from the work function
// - Any error that occurred if retries are exhausted
func Call[T any](
	ctx context.Context,
	work func() (T, error),
	retryStrategy RetryStrategyFunc,
) (T, error) {
	var (
		result T
		err    error
	)

	// Fallback to the default exponential backoff strategy if none is provided
	if retryStrategy == nil {
		retryStrategy = GetStrategy(ctx)
	}

	// Start the retry loop
	for retryCount := 0; ; retryCount++ {
		// Execute the work function one time
		result, err = work()

		// No error: stop retrying and return the result
		if err == nil {
			return result, err
		}

		// Non-retryable error: stop retrying and return the error
		if ErrNonRetryable.Is(err) {
			return result, err
		}

		// Non-transient gRPC error: stop retrying and return the error
		status, isGRPCError := status.FromError(err)
		if isGRPCError && !slices.Contains(transientGRPCErrorCodes, status.Code()) {
			return result, err
		}

		// TODO_IN_THIS_PR: #PUC
		if !retryStrategy(ctx, retryCount) {
			return result, err
		}
	}
}

// GetStrategy retrieves the retry strategy from the context.
// - Returns the default exponential delay strategy if no strategy is found
// - Useful for setting a custom retry strategy in the context and retrieving it later
func GetStrategy(ctx context.Context) RetryStrategyFunc {
	strategy, ok := ctx.Value(RetryStrategyCtxKey).(RetryStrategyFunc)
	if !ok {
		return DefaultExponentialDelay
	}
	return strategy
}

// WithExponentialBackoffFn creates a retry strategy with exponential backoff.
//
// This function returns a RetryStrategyFunc that implements exponential backoff behavior:
// - Retries stop after reaching maxRetryCount
// - Initial delay starts at initialDelayMs
// - Delay doubles with each retry attempt (2^retryCount)
// - Delay is capped at maxDelayMs
// - Respects context cancellation
//
// Parameters:
//   - maxRetryCount: Maximum number of retry attempts allowed
//   - initialDelayMs: Starting delay in milliseconds
//   - maxDelayMs: Upper limit for delay in milliseconds
//
// Returns a RetryStrategyFunc that can be used with retry mechanisms.
func WithExponentialBackoffFn(
	maxRetryCount int,
	initialDelayMs int,
	maxDelayMs int,
) RetryStrategyFunc {
	return func(ctx context.Context, retryCount int) bool {
		// Stop retrying if we've reached the maximum retry count
		if retryCount >= maxRetryCount {
			return false
		}

		// Calculate delay using exponential backoff formula: initial * 2^retryCount
		backoffDelay := initialDelayMs * int(math.Pow(exponentialBackOffFactor, float64(retryCount)))

		// Cap the delay at the maximum allowed value
		delayMs := min(backoffDelay, maxDelayMs)

		// Create a timer channel that will signal when the delay period is over
		delayCh := time.After(time.Duration(delayMs) * time.Millisecond)

		// Wait for either context cancellation or delay completion
		for {
			select {
			case <-ctx.Done():
				// Context was canceled, abort retry attempt
				return false
			case <-delayCh:
				// Delay period completed, proceed with retry
				return true
			}
		}
	}
}
