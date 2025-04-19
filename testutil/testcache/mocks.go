package testcache

import "github.com/stretchr/testify/mock"

// MockKeyValueCache is a mock implementation of the KeyValueCache interface.
// DEV_NOTE: Since gomock does not support generics, we're using testify/mock instead here.
type MockKeyValueCache[T any] struct {
	mock.Mock
}

// MockParamsCache is a mock implementation of the ParamsCache interface.
// DEV_NOTE: Since gomock does not support generics, we're using testify/mock instead here.
type MockParamsCache[T any] struct {
	mock.Mock
}

func NewNoopKeyValueCache[T any]() *MockKeyValueCache[T] {
	var zeroT T
	cache := &MockKeyValueCache[T]{}
	// Always simulate a cache miss.
	cache.On("Get", mock.Anything).Return(zeroT, false)
	cache.On("Set", mock.Anything, mock.Anything)
	cache.On("Delete", mock.Anything)
	cache.On("Clear")

	return cache
}

func (m *MockKeyValueCache[T]) Get(key string) (T, bool) {
	args := m.Called(key)
	return args.Get(0).(T), args.Bool(1)
}

func (m *MockKeyValueCache[T]) Set(key string, value T) {
	m.Called(key, value)
}

func (m *MockKeyValueCache[T]) Delete(key string) {
	m.Called(key)
}

func (m *MockKeyValueCache[T]) Clear() {
	m.Called()
}

func NewNoopParamsCache[T any]() *MockParamsCache[T] {
	var zeroT T
	cache := &MockParamsCache[T]{}
	// Always simulate a cache miss.
	cache.On("Get", mock.Anything).Return(zeroT, false)
	cache.On("Set", mock.Anything, mock.Anything)
	cache.On("Delete", mock.Anything)
	cache.On("Clear")

	return cache
}

func (m *MockParamsCache[T]) Get() (T, bool) {
	args := m.Called()
	return args.Get(0).(T), args.Bool(1)
}

func (m *MockParamsCache[T]) Set(value T) {
	m.Called(value)
}

func (m *MockParamsCache[T]) Clear() {
	m.Called()
}
