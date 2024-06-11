package common

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// GenericOptionFunc is an option func which receives a context and K type object for configuration.
type GenericOptionFunc[K any] func(context.Context, K) context.Context

// WithBlockHash sets the initial block hash for the context and returns the updated context.
// The O generic type should be the concrete option function type, where K is the second
// argument of the option function.
func WithBlockHash[O GenericOptionFunc[K], K any](hash []byte) O {
	return func(ctx context.Context, _ K) context.Context {
		return SetBlockHash(ctx, hash)
	}
}

// SetBlockHash updates the block hash for the given context and returns the updated context.
func SetBlockHash(ctx context.Context, hash []byte) context.Context {
	return cosmostypes.UnwrapSDKContext(ctx).WithHeaderHash(hash)
}

// WithBlockHeight sets the initial block height for the context and returns the updated context.
// The O generic type should be the concrete option function type, where K is the second
// argument of the option function.
func WithBlockHeight[O GenericOptionFunc[K], K any](height int64) O {
	return func(ctx context.Context, _ K) context.Context {
		return SetBlockHeight(ctx, height)
	}
}

// SetBlockHeight updates the block height for the given context and returns the updated context.
func SetBlockHeight(ctx context.Context, height int64) context.Context {
	return cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(height)
}
