package events

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// ResetEventManager re-initializes the cosmos event manager in the context such
// that prior event emissions are cleared. It returns the context as both a stdlib
// and cosmos context types.
func ResetEventManager(ctx context.Context) (context.Context, cosmostypes.Context) {
	sdkCtx := cosmostypes.
		UnwrapSDKContext(ctx).
		WithEventManager(cosmostypes.NewEventManager())
	return sdkCtx, sdkCtx
}
