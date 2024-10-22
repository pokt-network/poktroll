package keeper

import (
	"context"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

// EndBlockerAutoUndelegateFromUnstakedGateways is called every block and handles
// Application auto-undelegating from unstaked gateways.
// TODO_BLOCKER: Gateway unstaking should be delayed until the current block's
// session end height to align with the application's pending undelegations.
func (k Keeper) EndBlockerAutoUndelegateFromUnstakedGateways(ctx sdk.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Get all the GatewayUnstaked events emitted in the block to avoid checking
	// each application's delegated gateways for unstaked gateways.
	unstakedGateways, err := k.getUnstakedGateways(ctx, currentHeight)
	if err != nil {
		return err
	}

	for _, unstakedGateway := range unstakedGateways {
		for _, applicationAddr := range unstakedGateway.DelegatingApplicationAddresses {
			application, found := k.GetApplication(ctx, applicationAddr)
			if !found {
				continue
			}
			gwIdx := slices.Index(application.DelegateeGatewayAddresses, unstakedGateway.Address)
			if gwIdx >= 0 {
				application.DelegateeGatewayAddresses = append(
					application.DelegateeGatewayAddresses[:gwIdx],
					application.DelegateeGatewayAddresses[gwIdx+1:]...,
				)
				// Record the pending undelegation for the application to allow any upcoming
				// proofs to get the application's ring signatures.
				k.recordPendingUndelegation(ctx, &application, unstakedGateway.GetAddress(), currentHeight)
				k.SetApplication(ctx, application)
			}
		}
	}

	return nil
}

// getUnstakedGateways returns the gateways which were unstaked in the given tx events.
func (k Keeper) getUnstakedGateways(ctx context.Context, currentHeight int64) (unstakedGateways []*gatewaytypes.Gateway, err error) {
	for _, gateway := range k.gatewayKeeper.GetAllGateways(ctx) {
		if gateway.UnstakeSessionEndHeight != 0 && gateway.UnstakeSessionEndHeight <= currentHeight {
			unstakedGateways = append(unstakedGateways, &gateway)
		}
	}

	return unstakedGateways, nil
}
