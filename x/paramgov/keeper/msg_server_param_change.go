package keeper

import (
	"context"

    "github.com/pokt-network/poktroll/x/paramgov/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)


func (k msgServer) ParamChange(goCtx context.Context,  msg *types.MsgParamChange) (*types.MsgParamChangeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // TODO: Handling the message
    _ = ctx

	return &types.MsgParamChangeResponse{}, nil
}
