package application

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/application/keeper"
)

func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	if err := k.EndBlockerProcessPendingUndelegations(ctx); err != nil {
		return err
	}
	if err := k.EndBlockerPruneExpiredDelegations(ctx); err != nil {
		return err
	}

	return nil
}
