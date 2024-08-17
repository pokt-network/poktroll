package keeper

import (
	"context"

    "github.com/pokt-network/poktroll/x/application/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)


func (k msgServer) TransferApplicationStake(goCtx context.Context,  msg *types.MsgTransferApplicationStake) (*types.MsgTransferApplicationStakeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // TODO: Handling the message
    _ = ctx

	return &types.MsgTransferApplicationStakeResponse{}, nil
}
