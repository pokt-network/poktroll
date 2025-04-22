package types

import (
	storetypes "cosmossdk.io/store/types"
)

// RecordIterator is an interface for iterating over generic records.
//
// It provides methods to:
// - Navigate through records
// - Access current record data
// - Check iterator validity
// - Clean up resources
//
// DEV_NOTE: This was great as a supplement to the native Cosmos SDK Iterator
// to embed hidden functionality such as unmarshalling records from bytes.
// Ref: https://docs.cosmos.network/main/learn/advanced/store
type RecordIterator[T any] interface {
	// Next advances the iterator to the next record.
	Next()
	// Value retrieves the current record.
	// IT returns the record of type T
	Value() (T, error)
	// Valid checks if the iterator can still be used.
	// Returns false when:
	// - Iterator has been exhausted
	// - Iterator has been closed
	Valid() bool
	// Key retrieves the current byte key.
	Key() []byte
	// Close releases resources held by the iterator.
	Close()
}

// DataRecordAccessor is a function that transforms raw byte data into typed records.
// It takes a byte slice as input and returns a typed object of type T and an error.
type DataRecordAccessor[T any] func(data []byte) (T, error)

// Enforce that recordIterator implements RecordIterator
var _ RecordIterator[any] = (*recordIterator[any])(nil)

// recordIterator implements RecordIterator for blockchain state records.
// It combines:
// - A low-level Cosmos SDK store iterator
// - A function to convert raw bytes into typed objects
type recordIterator[T any] struct {
	storeIter         storetypes.Iterator
	deserializeRecord DataRecordAccessor[T]
}

// Next advances the iterator to the next position.
func (ri *recordIterator[T]) Next() {
	ri.storeIter.Next()
}

// Value returns the current record.
// Process:
// 1. Gets raw bytes from the store iterator
// 2. Deserializes bytes into a typed object using the deserializer function
func (ri *recordIterator[T]) Value() (T, error) {
	rawBytes := ri.storeIter.Value()
	return ri.deserializeRecord(rawBytes)
}

// Close releases iterator resources.
// Should be called when iteration is complete.
func (ri *recordIterator[T]) Close() {
	ri.storeIter.Close()
}

// Key returns the current record's key as bytes.
func (ri *recordIterator[T]) Key() []byte {
	return ri.storeIter.Key()
}

// Valid checks if the iterator is still usable.
// Returns false when the iterator has been exhausted or closed.
func (ri *recordIterator[T]) Valid() bool {
	return ri.storeIter.Valid()
}

// NewRecordIterator creates a new RecordIterator instance.
// Parameters:
// - storeIter: The underlying store iterator
// - deserializeRecord: Function to convert byte data to typed objects
// Returns:
// - A configured recordIterator instance
// TODO_CONSIDERATION: Add the possibility to configure a filter function such that
// the iterator can skip certain records based on custom logic.
func NewRecordIterator[T any](
	storeIter storetypes.Iterator,
	deserializeRecord DataRecordAccessor[T],
) *recordIterator[T] {
	return &recordIterator[T]{
		storeIter:         storeIter,
		deserializeRecord: deserializeRecord,
	}
}
