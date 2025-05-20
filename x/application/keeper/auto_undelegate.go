package keeper

import (
	"slices"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

// EndBlockerAutoUndelegateFromUnbondingGateways is called every block and handles
// Application auto-undelegating from unbonding gateways that are no longer active.
func (k Keeper) EndBlockerAutoUndelegateFromUnbondingGateways(ctx cosmostypes.Context) error {
	logger := k.Logger().With("method", "AutoUndelegateFromUnbondingGateways")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Get all the gateways that are unbonding and have reached their unstake session end height.
	unbondingGateways := k.getInactiveUnbondingGateways(ctx)

	for _, unbondingGateway := range unbondingGateways {
		// Iterate over the delegations index to get the applications that are
		// delegating to the unbonding gateway.
		delegationIterator := k.GetDelegationsIterator(ctx, unbondingGateway.GetAddress())
		defer delegationIterator.Close()

		for ; delegationIterator.Valid(); delegationIterator.Next() {
			application, err := delegationIterator.Value()
			if err != nil {
				return err
			}

			// Get the index of the particular unbonding gateway from the list of gateways the app delegated to.
			gwIdx := slices.Index(application.DelegateeGatewayAddresses, unbondingGateway.GetAddress())
			if gwIdx < 0 {
				// If the delegation is referencing an application that is not delegating
				// to the gateway, log the error, remove the index entry but continue
				// to the next delegation.
				logger.Error("Gateway address not found in application delegatee addresses")
				k.removeApplicationDelegationIndex(ctx, unbondingGateway.GetAddress(), application.Address)
				continue
			}

			// Remove the unbonding gateway from the application's delegatee list.
			application.DelegateeGatewayAddresses = append(
				application.DelegateeGatewayAddresses[:gwIdx],
				application.DelegateeGatewayAddresses[gwIdx+1:]...,
			)

			// Record the pending undelegation for the application to allow any upcoming
			// proofs to get the application's ring signatures.
			k.recordPendingUndelegation(ctx, &application, unbondingGateway.GetAddress(), currentHeight)

			k.SetApplication(ctx, application)
		}
	}

	return nil
}

// getInactiveUnbondingGateways returns the gateways which are unbonding and are no longer active.
func (k Keeper) getInactiveUnbondingGateways(ctx cosmostypes.Context) []*gatewaytypes.Gateway {
	currentBlockHeight := ctx.BlockHeight()
	// TODO_IMPROVE: Add a GetAllUnbondingGatewaysIterator method to the gateway keeper
	// to avoid fetching all gateways.
	gateways := k.gatewayKeeper.GetAllGateways(ctx)

	unbondingGateways := make([]*gatewaytypes.Gateway, 0)
	for _, gateway := range gateways {
		if !gateway.IsActive(currentBlockHeight) {
			unbondingGateways = append(unbondingGateways, &gateway)
		}
	}

	return unbondingGateways
}
