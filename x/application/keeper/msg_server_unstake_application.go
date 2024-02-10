package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/application/types"
)

func (k msgServer) UnstakeApplication(goCtx context.Context, msg *types.MsgUnstakeApplication) (*types.MsgUnstakeApplicationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgUnstakeApplicationResponse{}, nil
}
