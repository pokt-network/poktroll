package store

// TODO_IN_THIS_COMMIT: godoc...
type KeyValueStore interface {
	Get(key []byte) []byte
	Has(key []byte) bool
	Set(key []byte, value []byte)
	Delete(key []byte)
}
