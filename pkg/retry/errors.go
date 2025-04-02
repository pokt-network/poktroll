package retry

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace = "retry"
	// ErrNonRetryable signals the retry strategy to stop retrying due to a non-retryable error.
	ErrNonRetryable = sdkerrors.Register(codespace, 1, "non-retryable error")
)
