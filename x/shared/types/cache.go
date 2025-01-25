package types

import "sync"

type Cache[K comparable, V any] struct {
	store   map[K]V
	cacheMu *sync.RWMutex
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	val, ok := c.store[key]
	return val, ok
}

func (c *Cache[K, V]) Set(key K, val V) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.store[key] = val
}

func (c *Cache[K, V]) Delete(key K) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	delete(c.store, key)
}

func (c *Cache[K, V]) Clear() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	clear(c.store)
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		store:   make(map[K]V),
		cacheMu: &sync.RWMutex{},
	}
}
