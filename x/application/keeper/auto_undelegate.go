package keeper

import (
	"slices"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

// EndBlockerAutoUndelegateFromUnbondingGateways is called every block and handles
// Application auto-undelegating from unbonding gateways that are no longer active.
func (k Keeper) EndBlockerAutoUndelegateFromUnbondingGateways(ctx cosmostypes.Context) error {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Get all the gateways that are unbonding and have reached their unstake session end height.
	unbondingGateways := k.getInactiveUnbondingGateways(ctx)

	// TODO_POST_MAINNET: Once delegating applications are indexed by gateway address,
	// this can be optimized to only check applications that have delegated to
	// unstaked gateways.
	for _, application := range k.GetAllApplications(ctx) {
		for _, unbondingGateway := range unbondingGateways {
			gwIdx := slices.Index(application.DelegateeGatewayAddresses, unbondingGateway.GetAddress())
			if gwIdx >= 0 {
				application.DelegateeGatewayAddresses = append(
					application.DelegateeGatewayAddresses[:gwIdx],
					application.DelegateeGatewayAddresses[gwIdx+1:]...,
				)
				// Record the pending undelegation for the application to allow any upcoming
				// proofs to get the application's ring signatures.
				k.recordPendingUndelegation(ctx, &application, unbondingGateway.GetAddress(), currentHeight)
			}
		}

		k.SetApplication(ctx, application)
	}

	return nil
}

// getInactiveUnbondingGateways returns the gateways which are unbonding and are no longer active.
func (k Keeper) getInactiveUnbondingGateways(ctx cosmostypes.Context) []*gatewaytypes.Gateway {
	currentBlockHeight := ctx.BlockHeight()
	gateways := k.gatewayKeeper.GetAllGateways(ctx)

	unbondingGateways := make([]*gatewaytypes.Gateway, 0)
	for _, gateway := range gateways {
		if !gateway.IsActive(currentBlockHeight) {
			unbondingGateways = append(unbondingGateways, &gateway)
		}
	}

	return unbondingGateways
}
