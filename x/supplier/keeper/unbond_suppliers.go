package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// EndBlockerUnbondSuppliers unbonds suppliers whose unbonding period has elapsed.
func (k Keeper) EndBlockerUnbondSuppliers(ctx context.Context) error {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Only process unbonding suppliers at the end of the session.
	if sharedtypes.IsSessionEndHeight(&sharedParams, currentHeight) {
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

		unbondingHeight := sharedtypes.GetSupplierUnbondingHeight(&sharedParams, &supplier)

		// If the unbonding height is ahead of the current height, the supplier
		// stays in the unbonding state.
		if unbondingHeight > currentHeight {
			continue
		}

		// Retrieve the owner address of the supplier.
		ownerAddress, err := cosmostypes.AccAddressFromBech32(supplier.OwnerAddress)
		if err != nil {
			logger.Error(fmt.Sprintf("could not parse the owner address %s", supplier.OwnerAddress))
			return err
		}

		// Retrieve the address of the supplier.
		supplierOperatorAddress, err := cosmostypes.AccAddressFromBech32(supplier.OperatorAddress)
		if err != nil {
			logger.Error(fmt.Sprintf("could not parse the operator address %s", supplier.OperatorAddress))
			return err
		}

		// If the supplier stake is 0 due to slashing, then do not move 0 coins
		// to its account.
		// Coin#IsPositive returns false if the coin is 0.
		if supplier.Stake.IsPositive() {
			// Send the coins from the supplier pool back to the supplier.
			if err = k.bankKeeper.SendCoinsFromModuleToAccount(
				ctx, suppliertypes.ModuleName, ownerAddress, []cosmostypes.Coin{*supplier.Stake},
			); err != nil {
				logger.Error(fmt.Sprintf(
					"could not send %s coins from module %s to account %s due to %s",
					supplier.Stake.String(), suppliertypes.ModuleName, ownerAddress, err,
				))
				return err
			}
		}

		// Remove the supplier from the store.
		k.RemoveSupplier(ctx, supplierOperatorAddress.String())
		logger.Info(fmt.Sprintf("Successfully removed the supplier: %+v", supplier))

		// Emit an event which signals that the supplier has sucessfully unbonded.
		event := &suppliertypes.EventSupplierUnbondingEnd{
			Supplier:        &supplier,
			UnbondingHeight: unbondingHeight,
		}
		if eventErr := sdkCtx.EventManager().EmitTypedEvent(event); eventErr != nil {
			logger.Error(fmt.Sprintf("failed to emit event: %+v; %s", event, eventErr))
		}
	}

	return nil
}
