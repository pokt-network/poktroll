package either

// AsyncError represents a value which could either be a synchronous error or
// an asynchronous error (sent through a channel). It wraps the more generic
// `Either` type specific for error channels.
type (
	AsyncError Either[chan error]
	Bytes      = Either[[]byte]
)
