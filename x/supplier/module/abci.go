package supplier

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/keeper"
)

// EndBlocker is called every block and handles supplier related updates.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	// TODO_IMPROVE: Add logs and/or telemetry on the number of unbonded suppliers.
	if err := k.EndBlockerUnbondSuppliers(ctx); err != nil {
		return err
	}

	return nil
}
