package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"pocket/x/application/types"
)

func (k msgServer) UnstakeApplication(goCtx context.Context, msg *types.MsgUnstakeApplication) (*types.MsgUnstakeApplicationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgUnstakeApplicationResponse{}, nil
}
