package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) ClaimMorseSupplier(goCtx context.Context, msg *types.MsgClaimMorseSupplier) (*types.MsgClaimMorseSupplierResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO_UPNEXT(@bryanchriswhite, #1034): Handling the message
	_ = ctx

	return &types.MsgClaimMorseSupplierResponse{}, nil
}
