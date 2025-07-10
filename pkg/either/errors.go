package either

// SyncErr creates an AsyncError either from a synchronous error.
// It wraps the Error into the left field (conventionally associated with the
// error value in the Either pattern) of the Either type. It casts the result
// to the AsyncError type.
func SyncErr(err error) AsyncError {
	return AsyncError(Error[chan error](err))
}

// AsyncErr creates an AsyncError from an error channel.
// It wraps the error channel into the right field (conventionally associated with
// successful values in the Either pattern) of the Either type.
func AsyncErr(errCh chan error) AsyncError {
	return AsyncError(Success[chan error](errCh))
}

// SyncOrAsyncError decomposes the AsyncError into its components, returning
// an error channel and a synchronous error. If the AsyncError represents a
// synchronous error, the error channel will be nil and vice versa.
func (soaErr AsyncError) SyncOrAsyncError() (chan error, error) {
	errCh, err := Either[chan error](soaErr).ValueOrError()
	return errCh, err
}

// IsSyncError checks if the AsyncError represents a synchronous error.
func (soaErr AsyncError) IsSyncError() bool {
	return Either[chan error](soaErr).IsError()
}

// IsAsyncError checks if the AsyncError represents an asynchronous error
// (sent through a channel).
func (soaErr AsyncError) IsAsyncError() bool {
	return Either[chan error](soaErr).IsSuccess()
}
