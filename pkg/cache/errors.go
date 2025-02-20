package cache

import "cosmossdk.io/errors"

const codesace = "client/query/cache"

var (
	ErrCacheMiss                   = errors.Register(codesace, 1, "cache miss")
	ErrHistoricalModeNotEnabled    = errors.Register(codesace, 2, "historical mode not enabled")
	ErrQueryCacheConfigValidation  = errors.Register(codesace, 3, "invalid query cache config")
	ErrCacheInternal               = errors.Register(codesace, 4, "cache internal error")
	ErrUnsupportedHistoricalModeOp = errors.Register(codesace, 5, "operation not supported in historical mode")
)
