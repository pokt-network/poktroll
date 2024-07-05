package supplier

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/keeper"
)

// EndBlocker is called every block and handles supplier related updates.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	if err := k.EndBlockerUnbondSupplier(ctx); err != nil {
		return err
	}

	return nil
}
