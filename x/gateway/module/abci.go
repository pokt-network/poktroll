package gateway

import (
	"fmt"

	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/pocket/x/gateway/keeper"
	"github.com/pokt-network/pocket/x/gateway/types"
)

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
