package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// NumSessionsAppToGatewayUndelegationRetention is the number of sessions for which
// undelegation from applications to gateways are delayed before being pruned.
// TODO_DOCUMENT(@red-0ne): Need to document the flow from this comment
// so its clear to everyone why this is necessary; https://github.com/pokt-network/poktroll/issues/476#issuecomment-2052639906.
// TODO_MAINNET(#516): Should this be configurable? Note that it should
// likely be a function of SubmitProofCloseWindowNumBlocks once implemented.
const NumSessionsAppToGatewayUndelegationRetention = 2

// EndBlockerPruneAppToGatewayPendingUndelegation runs at the end of each block
// and prunes app to gateway undelegations that have exceeded the retention delay.
func (k Keeper) EndBlockerPruneAppToGatewayPendingUndelegation(ctx sdk.Context) error {
	currentHeight := ctx.BlockHeight()

	// Calculate the block height at which undelegations should be pruned
	numBlocksUndelegationRetention := k.GetNumBlocksUndelegationRetention(ctx)
	if currentHeight <= numBlocksUndelegationRetention {
		return nil
	}
	earliestUnprunedUndelegationHeight := uint64(currentHeight - numBlocksUndelegationRetention)

	// Iterate over all applications and prune undelegations that are older than
	// the retention period.
	for _, application := range k.GetAllApplications(ctx) {
		for undelegationSessionEndHeight := range application.PendingUndelegations {
			if undelegationSessionEndHeight < earliestUnprunedUndelegationHeight {
				// prune undelegations
				delete(application.PendingUndelegations, undelegationSessionEndHeight)
			}
		}

		k.SetApplication(ctx, application)
	}

	return nil
}

// GetNumBlocksUndelegationRetention returns the number of blocks for which
// undelegations should be kept before being pruned, given the current on-chain
// shared module parameters.
func (k Keeper) GetNumBlocksUndelegationRetention(ctx context.Context) int64 {
	sharedParams := k.sharedKeeper.GetParams(ctx)
	return GetNumBlocksUndelegationRetention(&sharedParams)
}

// GetNumBlocksUndelegationRetention returns the number of blocks for which
// undelegations should be kept before being pruned, given the passed shared
// module parameters.
func GetNumBlocksUndelegationRetention(sharedParams *sharedtypes.Params) int64 {
	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())

	return shared.SessionGracePeriodBlocks +
		(numBlocksPerSession * NumSessionsAppToGatewayUndelegationRetention)
}
