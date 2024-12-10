package cache

import "cosmossdk.io/errors"

const codesace = "client/query/cache"

var (
	ErrCacheMiss                = errors.Register(codesace, 1, "cache miss")
	ErrHistoricalModeNotEnabled = errors.Register(codesace, 2, "historical mode not enabled")
)
