package either

import "reflect"

type Either[T any] struct {
	value T
	error error
}

func NewEither[T any](value T, error error) Either[T] {
	return Either[T]{value: value, error: error}
}

func Right[T any](value T) Either[T] {
	return Either[T]{value: value, error: nil}
}

func Left[T any](error error) Either[T] {
	// see: https://stackoverflow.com/questions/73864711/get-type-parameter-from-a-generic-struct-using-reflection
	var zeroT [0]T
	typeT := reflect.TypeOf(zeroT).Elem()
	zeroValue := reflect.Zero(typeT).Interface().(T)
	return Either[T]{value: zeroValue, error: error}
}

func (m Either[T]) ValueOrError() (T, error) {
	return m.value, m.error
}
