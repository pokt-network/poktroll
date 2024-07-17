package keeper

import (
	"fmt"
	"slices"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
	"github.com/pokt-network/poktroll/proto/types/gateway"
)

var eventGatewayUnstaked = proto.MessageName(new(gateway.EventGatewayUnstaked))

// EndBlockerAutoUndelegateFromUnstakedGateways is called every block and handles
// Application auto-undelegating from unstaked gateways.
// TODO_BLOCKER: Gateway unstaking should be delayed until the current block's
// session end height to align with the application's pending undelegations.
func (k Keeper) EndBlockerAutoUndelegateFromUnstakedGateways(ctx sdk.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Get all the GatewayUnstaked events emitted in the block to avoid checking
	// each application's delegated gateways for unstaked gateways.
	unstakedGatewayEvents := k.getUnstakeGatewayEvents(sdkCtx.EventManager().Events())

	// TODO_IMPROVE: Once delegating applications are indexed by gateway address,
	// this can be optimized to only check applications that have delegated to
	// unstaked gateways.
	for _, application := range k.GetAllApplications(ctx) {
		for _, unstakedGateway := range unstakedGatewayEvents {
			gwIdx := slices.Index(application.DelegateeGatewayAddresses, unstakedGateway)
			if gwIdx >= 0 {
				application.DelegateeGatewayAddresses = append(
					application.DelegateeGatewayAddresses[:gwIdx],
					application.DelegateeGatewayAddresses[gwIdx+1:]...,
				)
				// Record the pending undelegation for the application to allow any upcoming
				// proofs to get the application's ring signatures.
				k.recordPendingUndelegation(ctx, &application, unstakedGateway, currentHeight)
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
