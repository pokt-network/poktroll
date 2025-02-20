package cache

// KeyValueCache is a key/value store style interface for a cache of a single type.
// It is intended to be used to cache query responses (or derivatives thereof),
// where each key uniquely indexes the most recent query response.
type KeyValueCache[T any] interface {
	Get(key string) (T, error)
	Set(key string, value T) error
	Delete(key string)
	Clear()
}

// HistoricalKeyValueCache extends KeyValueCache to support getting and setting values
// at multiple heights for a given key.
type HistoricalKeyValueCache[T any] interface {
	KeyValueCache[T]
	// GetLatestVersion returns the value of the latest version for the given key.
	GetLatestVersion(key string) (T, error)
	// GetVersion retrieves the nearest value <= the specified version number.
	GetVersion(key string, version int64) (T, error)
	// SetVersion adds or updates a value at a specific version number.
	SetVersion(key string, value T, version int64) error
}
