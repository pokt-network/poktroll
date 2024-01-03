package retry

import (
	"context"
	"time"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

type RetryFunc func() chan error

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

			if retryCount >= retryLimit {
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
