package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

func (k msgServer) StakeGateway(
	goCtx context.Context,
	msg *types.MsgStakeGateway,
) (*types.MsgStakeGatewayResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"stake_gateway",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger().With("method", "StakeGateway")
	logger.Info(fmt.Sprintf("about to stake gateway with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Retrieve the address of the gateway
	gatewayAddress, err := sdk.AccAddressFromBech32(msg.Address)
	// NB: This SHOULD NEVER happen because msg.ValidateBasic() validates the address as bech32.
	if err != nil {
		// TODO_TECHDEBT(#384): determine whether to continue using cosmos logger for debug level.
		logger.Info(fmt.Sprintf("ERROR: could not parse address %q", msg.Address))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if the gateway already exists or not
	var coinsToEscrow sdk.Coin
	gateway, isGatewayFound := k.GetGateway(ctx, msg.Address)
	if !isGatewayFound {
		logger.Info(fmt.Sprintf("gateway not found; creating new gateway for address %q", msg.Address))
		gateway = k.createGateway(ctx, msg)
		coinsToEscrow = *msg.Stake
	} else {
		logger.Info(fmt.Sprintf("gateway found; about to try and update gateway for address %q", msg.Address))
		currGatewayStake := *gateway.Stake
		if err = k.updateGateway(ctx, &gateway, msg); err != nil {
			logger.Error(fmt.Sprintf("could not update gateway for address %q due to error %v", msg.Address, err))
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		coinsToEscrow, err = (*msg.Stake).SafeSub(currGatewayStake)
		if err != nil {
			return nil, status.Error(
				codes.InvalidArgument,
				types.ErrGatewayInvalidStake.Wrapf(
					"stake (%s) must be higher than previous stake (%s)",
					msg.Stake, currGatewayStake,
				).Error(),
			)
		}
		logger.Info(fmt.Sprintf("gateway is going to escrow an additional %+v coins", coinsToEscrow))
	}

	// MUST ALWAYS stake or upstake (> 0 delta).
	if coinsToEscrow.IsZero() {
		err = types.ErrGatewayInvalidStake.Wrapf("gateway %q must escrow more than 0 additional coins", msg.GetAddress())
		logger.Info(fmt.Sprintf("ERROR: %s", err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// MUST ALWAYS have at least minimum stake.
	minStake := k.GetParams(ctx).MinStake
	if msg.Stake.Amount.LT(minStake.Amount) {
		err = types.ErrGatewayInvalidStake.Wrapf("gateway %q must stake at least %s", msg.Address, minStake)
		logger.Info(fmt.Sprintf("ERROR: %s", err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Send the coins from the gateway to the staked gateway pool
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, gatewayAddress, types.ModuleName, []sdk.Coin{coinsToEscrow})
	if err != nil {
		// TODO_TECHDEBT(#384): determine whether to continue using cosmos logger for debug level.
		logger.Error(fmt.Sprintf("could not escrowed %v coins from %q to %q module account due to %v", coinsToEscrow, gatewayAddress, types.ModuleName, err))
		return nil, err
	}

	// Update the Gateway in the store
	k.SetGateway(ctx, gateway)
	logger.Info(fmt.Sprintf("Successfully updated stake for gateway: %+v", gateway))

	isSuccessful = true
	return &types.MsgStakeGatewayResponse{
		Gateway: &gateway,
	}, nil
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
	// Checks if the msg address is the same as the current owner
	if msg.Address != gateway.Address {
		return types.ErrGatewayUnauthorized.Wrapf("msg Address %q != gateway address %q", msg.Address, gateway.Address)
	}
	if msg.Stake == nil {
		return types.ErrGatewayInvalidStake.Wrapf("stake amount cannot be nil")
	}
	if msg.Stake.IsLTE(*gateway.Stake) {
		return types.ErrGatewayInvalidStake.Wrapf("stake amount %v must be higher than previous stake amount %v", msg.Stake, gateway.Stake)
	}
	gateway.Stake = msg.Stake
	return nil
}
