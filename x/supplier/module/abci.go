package supplier

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// EndBlocker is called every block and handles supplier related updates.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	// Telemetry: measure execution time like standard cosmos-sdk modules do that.
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyEndBlocker)

	// TODO_IMPROVE(@red-0ne): Add logs and/or telemetry on the number of unbonded suppliers.
	if err := k.EndBlockerUnbondSuppliers(ctx); err != nil {
		return err
	}

	return nil
}
