package keeper

import (
	"context"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/gateway/types"
)

func (k msgServer) StakeGateway(
	goCtx context.Context,
	msg *types.MsgStakeGateway,
) (*types.MsgStakeGatewayResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger().With("method", "StakeGateway")
	logger.Info(fmt.Sprintf("About to stake gateway with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Check if the gateway already exists or not
	var err error
	var coinsToDelegate sdk.Coin
	gateway, isGatewayFound := k.GetGateway(ctx, msg.Address)
	if !isGatewayFound {
		logger.Info(fmt.Sprintf("Gateway not found. Creating new gateway for address %s", msg.Address))
		gateway = k.createGateway(ctx, msg)
		coinsToDelegate = *msg.Stake
	} else {
		logger.Info(fmt.Sprintf("Gateway found. Updating gateway stake for address %s", msg.Address))
		currGatewayStake := *gateway.Stake
		if err = k.updateGateway(ctx, &gateway, msg); err != nil {
			return nil, err
		}
		coinsToDelegate = (*msg.Stake).Sub(currGatewayStake)
	}

	// Retrieve the address of the gateway
	gatewayAddress, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		logger.Error(fmt.Sprintf("could not parse address %s", msg.Address))
		return nil, err
	}

	// Send the coins from the gateway to the staked gateway pool
	err = k.bankKeeper.DelegateCoinsFromAccountToModule(ctx, gatewayAddress, types.ModuleName, []sdk.Coin{coinsToDelegate})
	if err != nil {
		logger.Error(fmt.Sprintf("could not send %v coins from %s to %s module account due to %v", coinsToDelegate, gatewayAddress, types.ModuleName, err))
		return nil, err
	}

	// Update the Gateway in the store
	k.SetGateway(ctx, gateway)
	logger.Info(fmt.Sprintf("Successfully updated stake for gateway: %+v", gateway))

	return &types.MsgStakeGatewayResponse{}, nil
}

func (k msgServer) createGateway(
	_ sdk.Context,
	msg *types.MsgStakeGateway,
) types.Gateway {
	return types.Gateway{
		Address: msg.Address,
		Stake:   msg.Stake,
	}
}

func (k msgServer) updateGateway(
	_ sdk.Context,
	gateway *types.Gateway,
	msg *types.MsgStakeGateway,
) error {
	// Checks if the the msg address is the same as the current owner
	if msg.Address != gateway.Address {
		return sdkerrors.Wrapf(types.ErrGatewayUnauthorized, "msg Address (%s) != gateway address (%s)", msg.Address, gateway.Address)
	}
	if msg.Stake == nil {
		return sdkerrors.Wrapf(types.ErrGatewayInvalidStake, "stake amount cannot be nil")
	}
	if msg.Stake.IsLTE(*gateway.Stake) {
		return sdkerrors.Wrapf(types.ErrGatewayInvalidStake, "stake amount %v must be higher than previous stake amount %v", msg.Stake, gateway.Stake)
	}
	gateway.Stake = msg.Stake
	return nil
}
