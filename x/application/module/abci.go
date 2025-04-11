package application

import (
	"fmt"

	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
)

// BeginBlocker is called every block and handles application params related updates that
// need to be effective at the start of the block.
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) error {
	// Telemetry: measure the begin-block execution time following standard cosmos-sdk practices.
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyBeginBlocker)

	logger := k.Logger().With("method", "BeginBlocker")

	effectiveParams, err := k.BeginBlockerActivateApplicationParams(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("could not activate application params due to error %v", err))
		return err
	}

	if effectiveParams != nil {
		logger.Info(fmt.Sprintf("activated new application params %v", effectiveParams))
	}

	return nil
}

// EndBlocker is called every block and handles application related updates.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	// Telemetry: measure the end-block execution time following standard cosmos-sdk practices.
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyEndBlocker)

	if err := k.EndBlockerAutoUndelegateFromUnbondingGateways(ctx); err != nil {
		return err
	}

	if err := k.EndBlockerPruneAppToGatewayPendingUndelegation(ctx); err != nil {
		return err
	}

	if err := k.EndBlockerUnbondApplications(ctx); err != nil {
		return err
	}

	if err := k.EndBlockerTransferApplication(ctx); err != nil {
		return err
	}

	return nil
}
