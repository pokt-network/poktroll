package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/gateway/types"
)

// TODO_TECHDEBT(#49): Add un-delegation from delegated apps
// TODO(#73): Determine if a gateway needs an unbonding period after unstaking.
func (k msgServer) UnstakeGateway(
	goCtx context.Context,
	msg *types.MsgUnstakeGateway,
) (*types.MsgUnstakeGatewayResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger(ctx).With("method", "UnstakeGateway")
	logger.Info("About to unstake gateway with msg: %v", msg)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Check if the gateway already exists or not
	var err error
	gateway, isGatewayFound := k.GetGateway(ctx, msg.Address)
	if !isGatewayFound {
		logger.Info("Gateway not found. Cannot unstake address %s", msg.Address)
		return nil, types.ErrGatewayNotFound
	}
	logger.Info("Gateway found. Unstaking gateway for address %s", msg.Address)

	// Retrieve the address of the gateway
	gatewayAddress, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		logger.Error("could not parse address %s", msg.Address)
		return nil, err
	}

	// Send the coins from the gateway pool back to the gateway
	err = k.bankKeeper.UndelegateCoinsFromModuleToAccount(ctx, types.ModuleName, gatewayAddress, []sdk.Coin{*gateway.Stake})
	if err != nil {
		logger.Error("could not send %v coins from %s module to %s account due to %v", gateway.Stake, gatewayAddress, types.ModuleName, err)
		return nil, err
	}

	// Update the Gateway in the store
	k.RemoveGateway(ctx, gatewayAddress.String())
	logger.Info("Successfully removed the gateway: %+v", gateway)
	return &types.MsgUnstakeGatewayResponse{}, nil
}
