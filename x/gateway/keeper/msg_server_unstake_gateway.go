package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

func (k msgServer) UnstakeGateway(goCtx context.Context, msg *types.MsgUnstakeGateway) (*types.MsgUnstakeGatewayResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgUnstakeGatewayResponse{}, nil
}
