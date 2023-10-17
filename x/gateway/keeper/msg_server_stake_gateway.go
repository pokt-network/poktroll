package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"pocket/x/gateway/types"
)

func (k msgServer) StakeGateway(goCtx context.Context, msg *types.MsgStakeGateway) (*types.MsgStakeGatewayResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgStakeGatewayResponse{}, nil
}
