package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) UploadMorseState(goCtx context.Context, msg *types.MsgUploadMorseState) (*types.MsgUploadMorseStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgUploadMorseStateResponse{}, nil
}
