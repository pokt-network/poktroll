package keeper

import (
	"slices"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

// EndBlockerAutoUndelegateFromUnstakedGateways is called every block and handles
// Application auto-undelegating from unstaked gateways.
// TODO_BLOCKER: Gateway unstaking should be delayed until the current block's
// session end height to align with the application's pending undelegations.
func (k Keeper) EndBlockerAutoUndelegateFromUnstakedGateways(ctx cosmostypes.Context) error {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Get all the GatewayUnstaked events emitted in the block to avoid checking
	// each application's delegated gateways for unstaked gateways.
	unstakedGateways, err := k.getUnstakedGateways(sdkCtx.EventManager().Events())
	if err != nil {
		return err
	}

	// TODO_IMPROVE: Once delegating applications are indexed by gateway address,
	// this can be optimized to only check applications that have delegated to
	// unstaked gateways.
	for _, application := range k.GetAllApplications(ctx) {
		for _, unstakedGateway := range unstakedGateways {
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

// getUnstakedGateways returns the gateways which were unstaked in the given tx events.
func (k Keeper) getUnstakedGateways(
	events cosmostypes.Events,
) (unstakedGateways []*gatewaytypes.Gateway, err error) {
	for _, e := range events.ToABCIEvents() {
		typedEvent, err := cosmostypes.ParseTypedEvent(e)
		if err != nil {
			// Ignore non-typed errors (e.g. coin_received).
			continue
		}

		// Ignore events which are not gateway unstaked events.
		gatewayUnstakedEvent, ok := typedEvent.(*gatewaytypes.EventGatewayUnstaked)
		if !ok {
			continue
		}

		unstakedGateways = append(unstakedGateways, gatewayUnstakedEvent.GetGateway())
	}

	return unstakedGateways, nil
}
