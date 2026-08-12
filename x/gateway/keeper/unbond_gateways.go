package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

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
	if !sharedtypes.IsSessionEndHeight(&sharedParams, currentHeight) {
		return numUnbondedGateways, nil
	}

	// Iterate over all gateways and unbond the ones that have finished the unbonding period.
	// TODO_POST_MAINNET: Use an index to iterate over the gateways that have initiated
	// the unbonding action instead of iterating over all of them.
	// Decode WITHOUT cards: this scan only needs lifecycle state, and a card can be
	// 256 KiB. See GetAllGatewayLifecycles.
	for _, gatewayLifecycle := range k.GetAllGatewayLifecycles(ctx) {
		// Skip over gateways that have not initiated the unbonding action since it's a no-op.
		if !gatewayLifecycle.IsUnbonding() {
			continue
		}
		gateway := gatewayLifecycle.ToGateway()

		// Compute the unbonding end height using the shared params effective when the gateway
		// began unbonding (its unstake session end height), NOT the live params, so a later
		// num_blocks_per_session decrease cannot release it early (#543, F1).
		unstakeParams := k.sharedKeeper.GetParamsAtHeight(ctx, int64(gateway.GetUnstakeSessionEndHeight()))
		unbondingEndHeight := gatewaytypes.GetGatewayUnbondingHeight(&unstakeParams, &gateway)

		// If the unbonding height is ahead of the current height, the gateway
		// stays in the unbonding state.
		if unbondingEndHeight > currentHeight {
			continue
		}

		// DEV_NOTE: Auto-undelegating applications from unbonding gateways is taken care
		// of by the application module's EndBlockerAutoUndelegateFromUnbondingGateways.
		if err := k.UnbondGateway(ctx, &gateway); err != nil {
			return numUnbondedGateways, err
		}

		sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
		unbondingEndEvent := &gatewaytypes.EventGatewayUnbondingEnd{
			Gateway:            gateway.DehydratedForEvent(),
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
		ctx, gatewaytypes.ModuleName, gatewayAddr, []cosmostypes.Coin{*gateway.Stake},
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
	// Identifying fields only -- a Gateway can carry a 256 KiB metadata card, and %+v
	// renders it inline (~317 KB per line). Today's caller passes a
	// card-less projection from GatewayLifecycle.ToGateway(), but UnbondGateway is
	// exported and takes a *Gateway, so this must not depend on that.
	logger.Info(fmt.Sprintf("Successfully removed the gateway: %s", gateway.GetAddress()))

	return nil
}
