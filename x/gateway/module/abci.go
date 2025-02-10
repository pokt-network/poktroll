package gateway

import (
	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/gateway/keeper"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

// EndBlocker is called every block and handles gateway related updates.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	// Telemetry: measure the end-block execution time following standard cosmos-sdk practices.
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyEndBlocker)

	if err := k.EndBlockerUnbondGateways(ctx); err != nil {
		return err
	}

	return nil
}
