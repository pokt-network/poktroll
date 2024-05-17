package either

import "github.com/pokt-network/poktroll/pkg/relayer"

type (
	// AsyncError represents a value which could either be a synchronous error or
	// an asynchronous error (sent through a channel). It wraps the more generic
	// `Either` type specific for error channels.
	AsyncError   Either[chan error]
	Bytes        = Either[[]byte]
	SessionTrees = Either[[]relayer.SessionTree]
)
