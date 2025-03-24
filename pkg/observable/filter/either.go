package filter

import (
	"context"

	"github.com/pokt-network/pocket/pkg/either"
	"github.com/pokt-network/pocket/pkg/observable"
	"github.com/pokt-network/pocket/pkg/observable/channel"
)

// EitherError operates on an observable of an either type. It filters for all
// eithers which are not populated with errors, and maps them to their errors
// (publishes errors to the resulting observable).
func EitherError[T any](
	ctx context.Context,
	eitherObservable observable.Observable[either.Either[T]],
) observable.Observable[error] {
	return channel.Map(
		ctx,
		eitherObservable,
		mapEitherError[T],
	)
}

// EitherSuccess operates on an observable of an either type. It filters for all
// eithers which are not populated with values, and maps them to their values
// (publishes values to the resulting observable).
func EitherSuccess[T any](
	ctx context.Context,
	eitherObservable observable.Observable[either.Either[T]],
) observable.Observable[T] {
	return channel.Map(
		ctx,
		eitherObservable,
		mapEitherSuccess[T],
	)
}

// mapEitherError is a MapFn that maps an either to its error. It skips the
// notification if the either is populated with a value.
func mapEitherError[T any](
	_ context.Context,
	inputEither either.Either[T],
) (_ error, skip bool) {
	if _, err := inputEither.ValueOrError(); err != nil {
		return err, false
	}
	return nil, true
}

// mapEitherSuccess is a MapFn that maps an either to its value. It skips the
// notification if the either is populated with an error.
func mapEitherSuccess[T any](
	_ context.Context,
	inputEither either.Either[T],
) (_ T, skip bool) {
	value, err := inputEither.ValueOrError()
	if err != nil {
		return value, true
	}
	return value, false
}
