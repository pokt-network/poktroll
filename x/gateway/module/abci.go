package gateway

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/gateway/keeper"
)

// EndBlocker is called every block and handles application related updates.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	if err := k.EndBlockerUnbondGateways(ctx); err != nil {
		return err
	}

	return nil
}
