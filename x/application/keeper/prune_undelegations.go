package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

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
	logger := k.Logger().With("method", "PruneAppToGatewayPendingUndelegation")

	currentHeight := ctx.BlockHeight()

	// Calculate the block height at which undelegations should be pruned
	numBlocksUndelegationRetention := k.GetNumBlocksUndelegationRetention(ctx)
	// Skip pruning when current height is less than retention pTargeteriod to prevent
	// looking up negative or zero block heights.
	if currentHeight <= numBlocksUndelegationRetention {
		return nil
	}
	earliestUnprunedUndelegationHeight := uint64(currentHeight - numBlocksUndelegationRetention)

	// Iterate over all applications that have pending undelegations and prune
	// undelegations that are older than the retention period.
	// ALL_UNDELEGATIONS is passed to retrieve all undelegations instead of
	// targeting a specific application.
	allUndelegationsIterator := k.GetUndelegationsIterator(ctx, ALL_UNDELEGATIONS)
	defer allUndelegationsIterator.Close()

	for ; allUndelegationsIterator.Valid(); allUndelegationsIterator.Next() {
		undelegation, err := allUndelegationsIterator.Value()
		if err != nil {
			return err
		}

		application, found := k.GetApplication(ctx, undelegation.ApplicationAddress)
		if !found {
			// If the undelegation is referencing an application that is not
			// found in the store, log the error, remove the index entry but continue
			// to the next undelegation.
			logger.Error(fmt.Sprintf(
				"application with address %s not found but is referenced in undelegation index",
				undelegation.ApplicationAddress,
			))
			k.removeApplicationUndelegationIndex(ctx, allUndelegationsIterator.Key())
			continue
		}

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
// undelegations should be kept before being pruned, given the current onchain
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

	return int64(sharedParams.GetGracePeriodEndOffsetBlocks()) +
		(numBlocksPerSession * NumSessionsAppToGatewayUndelegationRetention)
}
