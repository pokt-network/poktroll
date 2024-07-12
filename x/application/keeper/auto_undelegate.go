package keeper

import (
	"fmt"
	"slices"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const eventGatewayUnstaked = "poktroll.gateway.GatewayUnstaked"

// EndBlockerAutoUndelegateFromUnstakedGateways is called every block and handles
// Application auto-undelegating from unstaked gateways.
// TODO_BLOCKER: Gateway unstaking should be subject to an unbonding period that
// finishes at a session end height to align with application pending undelegations.
func (k Keeper) EndBlockerAutoUndelegateFromUnstakedGateways(ctx sdk.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get all the GatewayUnstaked events emitted in the block to avoid checking
	// each application's delegated gateways for unstaked gateways.
	unstakedGateways := k.getUnstakeGatewayEvents(sdkCtx.EventManager().Events())

	// TODO_IMPROVE: Once delegating applications are indexed by gateway address,
	// this can be optimized to only check applications that have delegated to
	// unstaked gateways.
	for _, application := range k.GetAllApplications(ctx) {
		// Collect the gateway addresses that are still staked to avoid updating
		// the DelegatedGatewayAddresses's loop index while iterating.
		newDelegatedGatewayAddresses := make([]string, 0)
		for _, gatewayAddress := range application.DelegateeGatewayAddresses {
			if !slices.Contains(unstakedGateways, gatewayAddress) {
				newDelegatedGatewayAddresses = append(newDelegatedGatewayAddresses, gatewayAddress)
			}
		}
		application.DelegateeGatewayAddresses = newDelegatedGatewayAddresses

		for undelegationSessionEndHeight, pendingUndelegation := range application.PendingUndelegations {
			// Collect the gateway addresses that are still staked to avoid updating
			// the GatewayAddresses's loop index while iterating.
			newPendingUndelegations := make([]string, 0)
			for _, gatewayAddress := range pendingUndelegation.GatewayAddresses {
				if !slices.Contains(unstakedGateways, gatewayAddress) {
					newPendingUndelegations = append(newPendingUndelegations, gatewayAddress)
				}

			}
			pendingUndelegation.GatewayAddresses = newPendingUndelegations

			// Remove the pending undelegation entry if there are no more gateways
			// to undelegate.
			if len(pendingUndelegation.GatewayAddresses) == 0 {
				delete(application.PendingUndelegations, undelegationSessionEndHeight)
			}
		}

		k.SetApplication(ctx, application)
	}

	return nil
}

// getUnstakeGatewayEvents returns the addresses of the gateways that were
// unstaked in the block.
func (k Keeper) getUnstakeGatewayEvents(events sdk.Events) []string {
	unstakedGateways := make([]string, 0)
	for _, event := range events {
		if event.Type != eventGatewayUnstaked {
			continue
		}

		for _, attribute := range event.Attributes {
			if attribute.Key != "address" {
				continue
			}

			gatewayAddr, err := strconv.Unquote(attribute.Value)
			if err != nil {
				k.Logger().Error(fmt.Sprintf("could not unquote gateway address %s", attribute.Value))
				continue
			}
			unstakedGateways = append(unstakedGateways, gatewayAddr)
			break
		}
	}

	return unstakedGateways
}
