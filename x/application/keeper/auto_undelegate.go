package keeper

import (
	"slices"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

// EndBlockerAutoUndelegateFromUnstakedGateways is called every block and handles
// Application auto-undelegating from unstaked gateways.
// TODO_BETA(@bryanchriswhite): Gateway unstaking should be delayed until the current block's
// session end height to align with the application's pending undelegations.
func (k Keeper) EndBlockerAutoUndelegateFromUnstakedGateways(ctx cosmostypes.Context) error {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Get all the gateways that are unbonding and have reached their unstake session end height.
	unbondingGateways := k.getUnbondingGateways(ctx)

	// TODO_IMPROVE: Once delegating applications are indexed by gateway address,
	// this can be optimized to only check applications that have delegated to
	// unstaked gateways.
	for _, application := range k.GetAllApplications(ctx) {
		for _, unstakedGateway := range unbondingGateways {
			gwIdx := slices.Index(application.DelegateeGatewayAddresses, unstakedGateway.GetAddress())
			if gwIdx >= 0 {
				application.DelegateeGatewayAddresses = append(
					application.DelegateeGatewayAddresses[:gwIdx],
					application.DelegateeGatewayAddresses[gwIdx+1:]...,
				)
				// Record the pending undelegation for the application to allow any upcoming
				// proofs to get the application's ring signatures.
				k.recordPendingUndelegation(ctx, &application, unstakedGateway.GetAddress(), currentHeight)
			}
		}

		k.SetApplication(ctx, application)
	}

	return nil
}

// getUnbondingGateways returns the gateways which are unbonding and have reached their
// unstake session end height.
func (k Keeper) getUnbondingGateways(ctx cosmostypes.Context) []*gatewaytypes.Gateway {
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
