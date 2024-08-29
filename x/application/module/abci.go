package application

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/application/keeper"
)

// EndBlocker is called every block and handles application related updates.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	if err := k.EndBlockerAutoUndelegateFromUnstakedGateways(ctx); err != nil {
		return err
	}

	if err := k.EndBlockerPruneAppToGatewayPendingUndelegation(ctx); err != nil {
		return err
	}

	if err := k.EndBlockerUnbondApplications(ctx); err != nil {
		return err
	}

	return nil
}
