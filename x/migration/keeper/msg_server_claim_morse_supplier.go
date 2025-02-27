package keeper

import (
	"context"

    "github.com/pokt-network/poktroll/x/migration/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)


func (k msgServer) ClaimMorseSupplier(goCtx context.Context,  msg *types.MsgClaimMorseSupplier) (*types.MsgClaimMorseSupplierResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // TODO: Handling the message
    _ = ctx

	return &types.MsgClaimMorseSupplierResponse{}, nil
}
