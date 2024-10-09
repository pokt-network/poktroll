package application

import (
	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
)

// EndBlocker is called every block and handles application related updates.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyEndBlocker)

	if err := k.EndBlockerAutoUndelegateFromUnstakedGateways(ctx); err != nil {
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
