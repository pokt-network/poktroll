package mocks

// DEV_NOTE: This file was manually implemented (rather than using mockgen)
// to provide a common default behaviour in tests.

import (
	"encoding/binary"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ sharedtypes.RecordIterator[any] = (*sliceIterator[any])(nil)

// sliceIterator is a mock implementation of the RecordIterator interface.
// It is used for testing purposes and iterates over a slice of records.
type sliceIterator[T any] struct {
	records []T
	index   int
}

// Next moves the iterator to the next record.
func (ri *sliceIterator[T]) Next() {
	ri.index++
}

// Value retrieves the current record that the iterator is pointing to.
func (ri *sliceIterator[T]) Value() (T, error) {
	if ri.index < 0 || ri.index >= len(ri.records) {
		var zero T
		return zero, nil
	}
	return ri.records[ri.index], nil
}

// Valid checks if the iterator is still valid and can be used.
func (ri *sliceIterator[T]) Valid() bool {
	return ri.index >= 0 && ri.index < len(ri.records)
}

// Key retrieves the current key that the iterator is pointing to.
func (ri *sliceIterator[T]) Key() []byte {
	// Convert the index to a byte slice using binary.BigEndian
	buf := make([]byte, 8) // uint64 requires 8 bytes
	binary.BigEndian.PutUint64(buf, uint64(ri.index))
	return buf
}

// Close releases any resources held by the iterator.
func (ri *sliceIterator[T]) Close() {
	// No resources to release in this mock implementation
}

// NewMockRecordIterator creates a new mock record iterator with the given records.
// DEV_NOTE: Since gomock does not support generics, we're using testify/mock instead here.
func NewMockRecordIterator[T any](records []T) sharedtypes.RecordIterator[T] {
	return &sliceIterator[T]{
		records: records,
		index:   0,
	}
}
