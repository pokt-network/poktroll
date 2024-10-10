package events

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// ResetEventManager re-initializes the cosmos event manager in the context
// such that prior event emissions are cleared.
func ResetEventManager(ctx context.Context) context.Context {
	return cosmostypes.UnwrapSDKContext(ctx).
		WithEventManager(cosmostypes.NewEventManager())
}
