package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

// numSessionsAppToGatewayUndelegationRetention is the number of sessions for which
// undelegation from applications to gateways are delayed before being pruned.
// TODO_DOCUMENT(@red-0ne): Need to document the flow from this comment
// so its clear to everyone why this is necessary; https://github.com/pokt-network/poktroll/issues/476#issuecomment-2052639906.
// TODO_CONSIDERATION(#516): Should this be configurable? Note that it should
// likely be a function of SubmitProofCloseWindowNumBlocks once implemented.
const numSessionsAppToGatewayUndelegationRetention = 2

// EndBlockerPruneAppToGatewayPendingUndelegation runs at the end of each block
// and prunes app to gateway undelegations that have exceeded the retention delay.
func (k Keeper) EndBlockerPruneAppToGatewayPendingUndelegation(ctx sdk.Context) error {
	currentHeight := ctx.BlockHeight()

	// Calculate the block height at which undelegations should be pruned
	numBlocksUndelegationRetention := sessionkeeper.GetSessionGracePeriodBlockCount() +
		(sessionkeeper.NumBlocksPerSession * numSessionsAppToGatewayUndelegationRetention)
	pruningBlockHeight := uint64(currentHeight - numBlocksUndelegationRetention)

	// Iterate over all applications and prune undelegations that are older than
	// the retention period.
	for _, application := range k.GetAllApplications(ctx) {
		for undelegationSessionEndHeight := range application.PendingUndelegations {
			if undelegationSessionEndHeight > pruningBlockHeight {
				// prune undelegations
				delete(application.PendingUndelegations, undelegationSessionEndHeight)
			}
		}

		k.SetApplication(ctx, application)
	}

	return nil
}
