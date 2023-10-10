package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"pocket/x/supplier/types"
)

func (k msgServer) UnstakeSupplier(goCtx context.Context, msg *types.MsgUnstakeSupplier) (*types.MsgUnstakeSupplierResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgUnstakeSupplierResponse{}, nil
}
