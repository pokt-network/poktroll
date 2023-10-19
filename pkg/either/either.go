package either

// Either represents a type that can hold either a value of type T or an error.
// It's commonly used in functional programming to encapsulate functions that might fail.
// This way, instead of returning a value and an error separately, both are combined into a single type.
type Either[R any] struct {
	right R     // conventionally holds the success value
	left  error // conventionally holds the error, if any
}

func NewEither[R any](right R, left error) Either[R] {
	return Either[R]{right: right, left: left}
}

// Success creates a successful Either value.
func Success[T any](value T) Either[T] {
	return Either[T]{right: value, left: nil}
}

// Error creates an error Either value.
func Error[T any](err error) Either[T] {
	return Either[T]{right: zeroValue[T](), left: err}
}

// IsSuccess checks if the Either contains a success value.
func (m Either[T]) IsSuccess() bool {
	return m.left == nil
}

// IsError checks if the Either contains an error.
func (m Either[T]) IsError() bool {
	return m.left != nil
}

// ValueOrError unpacks the Either value.
func (m Either[T]) ValueOrError() (T, error) {
	return m.right, m.left
}

// Helper function to get the zero value for any type.
func zeroValue[T any]() (zero T) {
	return zero
}
