package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// EndBlockerUnbondGateways unbonds gateways whose unbonding period has elapsed.
func (k Keeper) EndBlockerUnbondGateways(ctx context.Context) (numUnbondedGateways int, err error) {
	logger := k.Logger().With("method", "EndBlockerUnbondGateways")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(sdkCtx)
	currentHeight := sdkCtx.BlockHeight()

	// Only process unbonding gateways at the end of the session.
	if sharedtypes.IsSessionEndHeight(&sharedParams, currentHeight) {
		return numUnbondedGateways, nil
	}

	// Iterate over all gateways and unbond the ones that have finished the unbonding period.
	// TODO_POST_MAINNET: Use an index to iterate over the gateways that have initiated
	// the unbonding action instead of iterating over all of them.
	for _, gateway := range k.GetAllGateways(ctx) {
		// Skip over gateways that have not initiated the unbonding action since it's a no-op.
		if !gateway.IsUnbonding() {
			continue
		}

		unbondingEndHeight := gatewaytypes.GetGatewayUnbondingHeight(&sharedParams, &gateway)

		// If the unbonding height is ahead of the current height, the gateway
		// stays in the unbonding state.
		if unbondingEndHeight > currentHeight {
			continue
		}

		if err := k.UnbondGateway(ctx, &gateway); err != nil {
			return numUnbondedGateways, err
		}

		sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
		unbondingEndEvent := &gatewaytypes.EventGatewayUnbondingEnd{
			Gateway:            &gateway,
			SessionEndHeight:   sessionEndHeight,
			UnbondingEndHeight: unbondingEndHeight,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(unbondingEndEvent); err != nil {
			err = gatewaytypes.ErrGatewayEmitEvent.Wrapf("(%+v): %s", unbondingEndEvent, err)
			logger.Error(err.Error())
			return numUnbondedGateways, err
		}

		numUnbondedGateways += 1
	}

	return numUnbondedGateways, nil
}

// UnbondGateway transfers the gateway stake to the bank module balance for the
// corresponding account and removes the gateway from the gateway module state.
func (k Keeper) UnbondGateway(ctx context.Context, gateway *gatewaytypes.Gateway) error {
	logger := k.Logger().With("method", "UnbondGateway")

	// Retrieve the account address of the gateway.
	gatewayAddr, err := cosmostypes.AccAddressFromBech32(gateway.Address)
	if err != nil {
		logger.Error(fmt.Sprintf("could not parse address %s", gateway.Address))
		return err
	}

	// Send the coins from the gateway pool back to the gateway.
	err = k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, gatewaytypes.ModuleName, gatewayAddr, []sdk.Coin{*gateway.Stake},
	)
	if err != nil {
		logger.Error(fmt.Sprintf(
			"could not send %v coins from module %s to gateway account %s due to %v",
			gateway.Stake, gatewayAddr, gatewaytypes.ModuleName, err,
		))
		return err
	}

	// Remove the Gateway from the store.
	k.RemoveGateway(ctx, gateway.GetAddress())
	logger.Info(fmt.Sprintf("Successfully removed the gateway: %+v", gateway))

	return nil
}
