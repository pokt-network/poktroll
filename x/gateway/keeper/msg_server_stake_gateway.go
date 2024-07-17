package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/proto/types/gateway"
	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

func (k msgServer) StakeGateway(
	goCtx context.Context,
	msg *gateway.MsgStakeGateway,
) (*gateway.MsgStakeGatewayResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"stake_gateway",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger().With("method", "StakeGateway")
	logger.Info(fmt.Sprintf("About to stake gateway with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Check if the gateway already exists or not
	var err error
	var coinsToEscrow sdk.Coin
	foundGateway, isGatewayFound := k.GetGateway(ctx, msg.Address)
	if !isGatewayFound {
		logger.Info(fmt.Sprintf("Gateway not found. Creating new gateway for address %q", msg.Address))
		foundGateway = k.createGateway(ctx, msg)
		coinsToEscrow = *msg.Stake
	} else {
		logger.Info(fmt.Sprintf("Gateway found. About to try and update gateway for address %q", msg.Address))
		currGatewayStake := *foundGateway.Stake
		if err = k.updateGateway(ctx, &foundGateway, msg); err != nil {
			logger.Error(fmt.Sprintf("could not update gateway for address %q due to error %v", msg.Address, err))
			return nil, err
		}
		coinsToEscrow, err = (*msg.Stake).SafeSub(currGatewayStake)
		if err != nil {
			return nil, err
		}
		logger.Info(fmt.Sprintf("Gateway is going to escrow an additional %+v coins", coinsToEscrow))
	}

	// Must always stake or upstake (> 0 delta)
	if coinsToEscrow.IsZero() {
		logger.Warn(fmt.Sprintf("Gateway %q must escrow more than 0 additional coins", msg.Address))
		return nil, gateway.ErrGatewayInvalidStake.Wrapf("gateway %q must escrow more than 0 additional coins", msg.Address)
	}

	// Retrieve the address of the gateway
	gatewayAddress, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		// TODO_TECHDEBT(#384): determine whether to continue using cosmos logger for debug level.
		logger.Error(fmt.Sprintf("could not parse address %q", msg.Address))
		return nil, err
	}

	// Send the coins from the gateway to the staked gateway pool
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, gatewayAddress, types.ModuleName, []sdk.Coin{coinsToEscrow})
	if err != nil {
		// TODO_TECHDEBT(#384): determine whether to continue using cosmos logger for debug level.
		logger.Error(fmt.Sprintf("could not escrowed %v coins from %q to %q module account due to %v", coinsToEscrow, gatewayAddress, types.ModuleName, err))
		return nil, err
	}

	// Update the Gateway in the store
	k.SetGateway(ctx, foundGateway)
	logger.Info(fmt.Sprintf("Successfully updated stake for gateway: %+v", foundGateway))

	isSuccessful = true
	return &gateway.MsgStakeGatewayResponse{}, nil
}

func (k msgServer) createGateway(
	_ sdk.Context,
	msg *gateway.MsgStakeGateway,
) gateway.Gateway {
	return gateway.Gateway{
		Address: msg.Address,
		Stake:   msg.Stake,
	}
}

func (k msgServer) updateGateway(
	_ sdk.Context,
	gatewayToUpdate *gateway.Gateway,
	msg *gateway.MsgStakeGateway,
) error {
	// Checks if the the msg address is the same as the current owner
	if msg.Address != gatewayToUpdate.Address {
		return gateway.ErrGatewayUnauthorized.Wrapf("msg Address %q != gateway address %q", msg.Address, gatewayToUpdate.Address)
	}
	if msg.Stake == nil {
		return gateway.ErrGatewayInvalidStake.Wrapf("stake amount cannot be nil")
	}
	if msg.Stake.IsLTE(*gatewayToUpdate.Stake) {
		return gateway.ErrGatewayInvalidStake.Wrapf("stake amount %v must be higher than previous stake amount %v", msg.Stake, gatewayToUpdate.Stake)
	}
	gatewayToUpdate.Stake = msg.Stake
	return nil
}
