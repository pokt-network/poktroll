package cache

import "cosmossdk.io/errors"

const codesace = "client/query/cache"

var (
	ErrKeyValueCacheConfigValidation = errors.Register(codesace, 3, "invalid query cache config")
	ErrCacheInternal                 = errors.Register(codesace, 4, "cache internal error")
)
