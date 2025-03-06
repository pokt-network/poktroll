package testcache

import "github.com/stretchr/testify/mock"

type MockKeyValueCache[T any] struct {
	mock.Mock
}

func NewNoopKeyValueCache[T any]() *MockKeyValueCache[T] {
	var zeroT T
	cache := &MockKeyValueCache[T]{}
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
