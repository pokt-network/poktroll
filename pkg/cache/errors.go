package cache

import "cosmossdk.io/errors"

const codespace = "cache"

var (
	ErrKeyValueCacheConfigValidation = errors.Register(codespace, 1, "invalid query cache config")
	ErrCacheInternal                 = errors.Register(codespace, 2, "cache internal error")
	ErrNoOverwrite                   = errors.Register(codespace, 3, "refusing to overwrite existing value")
)
