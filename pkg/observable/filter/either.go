package filter

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
)

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

func mapEitherError[T any](
	_ context.Context,
	inputEither either.Either[T],
) (_ error, skip bool) {
	if _, err := inputEither.ValueOrError(); err != nil {
		return err, false
	}
	return nil, true
}

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
