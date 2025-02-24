package cache

// KeyValueCache is a key/value store style interface for a cache of a single type.
// It is intended to be used to cache arbitrary data, where each key uniquely indexes
// the most recently observed version of the data associated with that key.
type KeyValueCache[T any] interface {
	Get(key string) (T, bool)
	Set(key string, value T)
	Delete(key string)
	Clear()
}

// HistoricalKeyValueCache is a key/value store style interface for a cache of a single type.
// It is intended to be used to cache arbitrary data, where each key uniquely indexes
// a mapping of version numbers to values corresponding to the historical values of the data
// associated with that key.
type HistoricalKeyValueCache[T any] interface {
	// GetLatestVersion returns the value of the latest version for the given key.
	GetLatestVersion(key string) (T, bool)

	// GetVersion retrieves the value at the specified version number.
	// It returns true if the value is cached, false otherwise.
	GetVersion(key string, version int64) (T, bool)

	// GetVersionLTE retrieves the value at the nearest version <= maxVersion.
	// It returns true if a value meeting the version criteria is cached, false otherwise.
	GetVersionLTE(key string, maxVersion int64) (T, bool)

	// SetVersion adds or updates a value at a specific version number.
	SetVersion(key string, value T, version int64) error
}
