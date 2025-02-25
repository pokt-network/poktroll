package query

// ParamsCache is an interface for a simple in-memory cache implementation for query parameters.
// It does not involve key-value pairs, but only stores a single value.
type ParamsCache[T any] interface {
	Get() (T, bool)
	Set(T)
	Clear()
}

// KeyValueCache is an interface for a simple in-memory key-value cache implementation.
type KeyValueCache[V any] interface {
	Get(string) (V, bool)
	Set(string, V)
	Delete(string)
	Clear()
}
