package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"pocket/x/application/types"
)

func (k msgServer) UndelegateFromGateway(goCtx context.Context, msg *types.MsgUndelegateFromGateway) (*types.MsgUndelegateFromGatewayResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger(ctx).With("method", "UndelegateFromGateway")
	logger.Info("About to undelegate application from gateway with msg: %v", msg)

	if err := msg.ValidateBasic(); err != nil {
		logger.Error("Undelegation Message failed basic validation: %v", err)
		return nil, err
	}

	// Undelegate the application from the gateway
	if err := k.UndelegateGateway(ctx, msg.AppAddress, msg.GatewayAddress); err != nil {
		return nil, err
	}

	return &types.MsgUndelegateFromGatewayResponse{}, nil
}
