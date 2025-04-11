package gateway

import (
	"fmt"

	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/gateway/keeper"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

// BeginBlocker is called every block and handles gateway params related updates that
// need to be effective at the start of the block.
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) error {
	// Telemetry: measure the begin-block execution time following standard cosmos-sdk practices.
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyBeginBlocker)

	logger := k.Logger().With("method", "BeginBlocker")

	effectiveParams, err := k.BeginBlockerActivateGatewayParams(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("could not activate gateway params due to error %v", err))
		return err
	}

	if effectiveParams != nil {
		logger.Info(fmt.Sprintf("activated new gateway params %v", effectiveParams))
	}

	return nil
}

// EndBlocker is called every block and handles gateway related updates.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	// Telemetry: measure the end-block execution time following standard cosmos-sdk practices.
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyEndBlocker)

	logger := k.Logger().With("method", "EndBlocker")

	numUnbondedGateways, err := k.EndBlockerUnbondGateways(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("could not unbond gateways due to error %v", err))
		return err
	}

	logger.Info(fmt.Sprintf(
		"unbonded %d gateways",
		numUnbondedGateways,
	))

	return nil
}
