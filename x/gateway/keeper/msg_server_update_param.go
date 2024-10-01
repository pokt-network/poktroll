package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/gateway/types"
)

func (k msgServer) UpdateParam(goCtx context.Context, msg *types.MsgUpdateParam) (*types.MsgUpdateParamResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgUpdateParamResponse{}, nil
}
