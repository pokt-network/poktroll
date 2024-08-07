package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/shared"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// EndBlockerUnbondSuppliers unbonds suppliers whose unbonding period has elapsed.
func (k Keeper) EndBlockerUnbondSuppliers(ctx context.Context) error {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Only process unbonding suppliers at the end of the session.
	if currentHeight != k.sharedKeeper.GetSessionEndHeight(ctx, currentHeight) {
		return nil
	}

	logger := k.Logger().With("method", "UnbondSupplier")

	// Iterate over all suppliers and unbond suppliers that have finished the unbonding period.
	// TODO_IMPROVE: Use an index to iterate over suppliers that have initiated the
	// unbonding action instead of iterating over all suppliers.
	for _, supplier := range k.GetAllSuppliers(ctx) {
		// Ignore suppliers that have not initiated the unbonding action.
		if !supplier.IsUnbonding() {
			continue
		}

		unbondingHeight := shared.GetSupplierUnbondingHeight(&sharedParams, &supplier)

		// If the unbonding height is ahead of the current height, the supplier
		// stays in the unbonding state.
		if unbondingHeight > currentHeight {
			continue
		}

		// Retrieve the address of the supplier.
		supplierAddress, err := cosmostypes.AccAddressFromBech32(supplier.Address)
		if err != nil {
			logger.Error(fmt.Sprintf("could not parse address %s", supplier.Address))
			return err
		}

		// Send the coins from the supplier pool back to the supplier.
		if err = k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, supplierAddress, []cosmostypes.Coin{*supplier.Stake},
		); err != nil {
			logger.Error(fmt.Sprintf(
				"could not send %s coins from %s module to %s account due to %s",
				supplier.Stake.String(), supplierAddress, types.ModuleName, err,
			))
			return err
		}

		// Remove the supplier from the store.
		k.RemoveSupplier(ctx, supplierAddress.String())
		logger.Info(fmt.Sprintf("Successfully removed the supplier: %+v", supplier))
	}

	return nil
}
