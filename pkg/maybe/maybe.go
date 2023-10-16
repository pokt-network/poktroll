package types

import "reflect"

type Maybe[T any] struct {
	value T
	error error
}

func NewMaybe[T any](value T, error error) Maybe[T] {
	return Maybe[T]{value: value, error: error}
}

func Just[T any](value T) Maybe[T] {
	return Maybe[T]{value: value, error: nil}
}

func JustError[T any](error error) Maybe[T] {
	// see: https://stackoverflow.com/questions/73864711/get-type-parameter-from-a-generic-struct-using-reflection
	var zeroT [0]T
	typeT := reflect.TypeOf(zeroT).Elem()
	zeroValue := reflect.Zero(typeT).Interface().(T)
	return Maybe[T]{value: zeroValue, error: error}
}

func (m Maybe[T]) ValueOrError() (T, error) {
	return m.value, m.error
}
