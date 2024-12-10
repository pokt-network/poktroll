package cache

// TODO_UP_NEXT(@bryanchriswhite): Implement a persistent cache using badger.
//
// var _ query.QueryCache[any] = (*BadgerCache[any])(nil)

// BadgerCache is a persistent cache backed by a badger database.
type BadgerCache[T any] struct {
	// db     *badger.DB
	// config CacheConfig
}
