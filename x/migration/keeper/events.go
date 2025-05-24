package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	types2 "github.com/pokt-network/poktroll/x/migration/types"
)

// emitEvents emits the given events via the event manager.
func emitEvents(ctx context.Context, events []types.Msg) error {
	sdkCtx := types.UnwrapSDKContext(ctx)
	for _, event := range events {
		if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
			return status.Error(
				codes.Internal,
				types2.ErrMorseSupplierClaim.Wrapf(
					"failed to emit event type %T: %v",
					event, err,
				).Error(),
			)
		}
	}
	return nil
}
