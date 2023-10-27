package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"pocket/x/application/types"
)

func (k msgServer) DelegateToGateway(goCtx context.Context, msg *types.MsgDelegateToGateway) (*types.MsgDelegateToGatewayResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// TODO: Handling the message
	_ = ctx

	return &types.MsgDelegateToGatewayResponse{}, nil
}
