package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"pocket/x/application/types"
)

func (k msgServer) StakeApplication(goCtx context.Context, msg *types.MsgStakeApplication) (*types.MsgStakeApplicationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgStakeApplicationResponse{}, nil
}
