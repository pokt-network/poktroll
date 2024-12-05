package cache

import "cosmossdk.io/errors"

const codesace = "client/query/cache"

var (
	// TODO_IN_THIS_COMMIT: godoc...
	ErrCacheMiss                = errors.Register(codesace, 1, "cache miss")
	ErrHistoricalModeNotEnabled = errors.Register(codesace, 2, "historical mode not enabled")
)
