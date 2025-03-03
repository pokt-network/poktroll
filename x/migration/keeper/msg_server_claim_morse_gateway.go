package keeper

import (
	"context"

    "github.com/pokt-network/poktroll/x/migration/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)


func (k msgServer) ClaimMorseGateway(goCtx context.Context,  msg *types.MsgClaimMorseGateway) (*types.MsgClaimMorseGatewayResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // TODO: Handling the message
    _ = ctx

	return &types.MsgClaimMorseGatewayResponse{}, nil
}
