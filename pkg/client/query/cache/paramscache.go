package cache

import (
	"sync"

	proto "github.com/cosmos/gogoproto/proto"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ query.ParamsCache[any] = (*paramsCache[any])(nil)

// paramsCache is a simple in-memory cache implementation for query parameters.
// It does not involve key-value pairs, but only stores a single value.
type paramsCache[T any] struct {
	logger  polylog.Logger
	cacheMu sync.RWMutex
	found   bool
	value   T
}

// NewParamsCache returns a new instance of a ParamsCache.
func NewParamsCache[T any](logger polylog.Logger) query.ParamsCache[T] {
	// Get the name of the cached type.
	cachedTypeName := "unknown"
	var zero T

	// Update the cached type name if the type is a proto message.
	if msg, ok := any(zero).(proto.Message); ok {
		cachedTypeName = proto.MessageName(msg)
	} else {
		logger.Warn().Msg("Could not determine cached type")
	}

	return &paramsCache[T]{
		logger: logger.With("type", cachedTypeName),
	}
}

// Get returns the value stored in the cache.
// A boolean is returned as the second value to indicate if the value was found in the cache.
func (c *paramsCache[T]) Get() (value T, found bool) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	if c.found {
		c.logger.Debug().Msg("Cache hit")
	}

	return c.value, c.found
}

// Set sets the value in the cache.
func (c *paramsCache[T]) Set(value T) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	c.found = true
	c.value = value
}

// Clear empties the cache.
func (c *paramsCache[T]) Clear() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	c.found = false

	var zero T
	c.value = zero
}
