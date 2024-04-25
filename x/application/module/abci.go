package application

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/application/keeper"
)

// EndBlocker called at every block to process pending undelegations and prune
// expired delegatee sets.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	// Process pending undelegations before pruning expired delegations since
	// undelegations involve archiving old delegations records which, depending on
	// the pruning parameters may result in the deletion of all archived records.

	if err := k.EndBlockerProcessPendingUndelegations(ctx); err != nil {
		return err
	}
	if err := k.EndBlockerPruneExpiredDelegations(ctx); err != nil {
		return err
	}

	return nil
}
