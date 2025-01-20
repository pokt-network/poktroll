package keeper

import (
	"context"

    "github.com/pokt-network/poktroll/x/migration/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)


func (k msgServer) UploadMorseState(goCtx context.Context,  msg *types.MsgUploadMorseState) (*types.MsgUploadMorseStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // TODO: Handling the message
    _ = ctx

	return &types.MsgUploadMorseStateResponse{}, nil
}
