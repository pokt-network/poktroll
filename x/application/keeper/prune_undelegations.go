package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

// undelegationRetentionSessions is the number of sessions for which undelegations
// are retained before being pruned.
// TODO_TECHDEBT: This should be a configuration parameter and potentially
// a governance parameter.
const undelegationRetentionSessions = 2

// EndBlockerPruneUndelegations runs at the end of each block and prunes
// undelegations that are older than the retention period.
func (k Keeper) EndBlockerPruneUndelegations(ctx sdk.Context) error {
	currentHeight := ctx.BlockHeight()
	// Calculate the block height at which undelegations should be pruned
	undelegationRetentionBlocks := sessionkeeper.GetSessionGracePeriodBlockCount() +
		(sessionkeeper.NumBlocksPerSession * undelegationRetentionSessions)

	// Iterate over all applications and prune undelegations that are older than
	// the retention period.
	for _, application := range k.GetAllApplications(ctx) {
		for undelegationsHeight := range application.Undelegations {
			if undelegationsHeight < uint64(currentHeight-undelegationRetentionBlocks) {
				// remove undelegations
				delete(application.Undelegations, undelegationsHeight)
			}
		}

		k.SetApplication(ctx, application)
	}

	return nil
}
