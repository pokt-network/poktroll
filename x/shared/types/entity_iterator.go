package types

import (
	storetypes "cosmossdk.io/store/types"
)

// RecordIterator is an interface that defines the methods for iterating over records
type RecordIterator[T any] interface {
	// Next moves the iterator to the next record.
	Next()
	// Value retrieves the current record that the iterator is pointing to.
	// It returns the record and an error if any occurred during retrieval.
	Value() (T, error)
	// Valid checks if the iterator is still valid and can be used.
	// Returns false when the iterator has been exhausted or closed.
	Valid() bool
	// Key retrieves the current key that the iterator is pointing to.
	Key() []byte
	// Close releases any resources held by the iterator.
	Close()
}

// IteratorRecordRetriever is a function type that retrieves a record from the store
// given a key. It takes a record key as input and returns a record of type T
type IteratorRecordRetriever[T any] func(key []byte) (T, error)

var _ RecordIterator[any] = (*recordIterator[any])(nil)

// recordIterator provides iteration functionality over records stored in the blockchain state.
// It wraps the underlying store iterator and a record retriever function to provide
// convenient access to onchain stored objects.
type recordIterator[T any] struct {
	iterator        storetypes.Iterator
	recordRetriever IteratorRecordRetriever[T]
}

// Next advances the iterator to the next position.
func (ri *recordIterator[T]) Next() {
	ri.iterator.Next()
}

// Value returns the current record that the iterator is pointing to. It:
// 1. Retrieves the value bytes from the iterator's current value
// 2. Uses the recordRetriever function to convert the bytes into a Record object
func (ri *recordIterator[T]) Value() (T, error) {
	valueBz := ri.iterator.Value()
	return ri.recordRetriever(valueBz)
}

// Close releases any resources held by the iterator.
// It should be called when the iterator is no longer needed.
func (ri *recordIterator[T]) Close() {
	ri.iterator.Close()
}

// Key returns the current key that the iterator is pointing to.
func (ri *recordIterator[T]) Key() []byte {
	return ri.iterator.Key()
}

// Valid returns whether the iterator is still valid and can continue to be used.
// Returns false when the iterator has been exhausted or closed.
func (ri *recordIterator[T]) Valid() bool {
	return ri.iterator.Valid()
}

// NewRecordIterator creates a new RecordIterator instance.
func NewRecordIterator[T any](
	iterator storetypes.Iterator,
	recordRetriever IteratorRecordRetriever[T],
) *recordIterator[T] {
	return &recordIterator[T]{
		iterator:        iterator,
		recordRetriever: recordRetriever,
	}
}
