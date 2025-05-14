package shared

import (
	"fmt"

	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/shared/keeper"
	"github.com/pokt-network/poktroll/x/shared/types"
)

// BeginBlocker is called every block and handles shared params related updates that
// need to be effective at the start of the block.
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) error {
	// Telemetry: measure the begin-block execution time following standard cosmos-sdk practices.
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyBeginBlocker)

	logger := k.Logger().With("method", "BeginBlocker")

	effectiveParams, err := k.BeginBlockerActivateSharedParams(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("could not activate shared params due to error %v", err))
		return err
	}

	if effectiveParams != nil {
		logger.Info(fmt.Sprintf("activated new shared params %v", effectiveParams))
	}

	return nil
}
