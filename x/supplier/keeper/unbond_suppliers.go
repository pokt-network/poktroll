package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// EndBlockerUnbondSuppliers unbonds suppliers whose unbonding period has elapsed.
func (k Keeper) EndBlockerUnbondSuppliers(ctx context.Context) (numUnbondedSuppliers uint64, err error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParamsUpdates := k.sharedKeeper.GetParamsUpdates(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Only process unbonding suppliers at the end of the session.
	if !sharedtypes.IsSessionEndHeight(sharedParamsUpdates, currentHeight) {
		return numUnbondedSuppliers, nil
	}

	logger := k.Logger().With("method", "UnbondSupplier")

	// Iterate over all unstaking suppliers and unbond suppliers that have finished the unbonding period.
	allUnstakingSuppliersIterator := k.GetAllUnstakingSuppliersIterator(ctx)
	defer allUnstakingSuppliersIterator.Close()

	for ; allUnstakingSuppliersIterator.Valid(); allUnstakingSuppliersIterator.Next() {
		supplierAddress := allUnstakingSuppliersIterator.Value()
		// Get dehydrated supplier from the store to avoid unmarshalling all the supplier service configs.
		supplier, found := k.GetDehydratedSupplier(ctx, string(supplierAddress))
		if !found {
			// We should be able to find the supplier if it is in the iterator.
			err := fmt.Errorf("should never happen: could not find unbonding supplier %s", supplierAddress)
			logger.Error(err.Error())
			return numUnbondedSuppliers, err
		}

		// Ignore suppliers that have not initiated the unbonding action
		// because this function is only responsible for unbonding.
		if !supplier.IsUnbonding() {
			// If we are getting the supplier from the unbonding store and it is not
			// unbonding, this means that there is a dangling entry in the index.
			// Log the error, remove the index entry but continue to the next supplier.
			err := fmt.Errorf("should never happen: found supplier %s in unbonding store but it is not unbonding", supplierAddress)
			logger.Error(err.Error())
			k.removeSupplierUnstakingHeightIndex(ctx, supplier.OperatorAddress)
			continue
		}

		unbondingEndHeight := sharedtypes.GetSupplierUnbondingEndHeight(sharedParamsUpdates, &supplier)

		// If the unbonding height is ahead of the current height, the supplier
		// stays in the unbonding state.
		if unbondingEndHeight > currentHeight {
			continue
		}

		// Retrieve the owner address of the supplier.
		ownerAddress, err := cosmostypes.AccAddressFromBech32(supplier.OwnerAddress)
		if err != nil {
			logger.Error(fmt.Sprintf("could not parse the owner address %s", supplier.OwnerAddress))
			return numUnbondedSuppliers, err
		}

		// Retrieve the address of the supplier.
		supplierOperatorAddress, err := cosmostypes.AccAddressFromBech32(supplier.OperatorAddress)
		if err != nil {
			logger.Error(fmt.Sprintf("could not parse the operator address %s", supplier.OperatorAddress))
			return numUnbondedSuppliers, err
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
				return numUnbondedSuppliers, err
			}
		}

		// TODO_CONSIDERATION: Should we hydrate the supplier service configurations
		// to expose the full supplier information to the event?
		// This can result in a lot of state bloat.
		// k.hydrateSupplierServiceConfigs(ctx, &supplier)

		// Remove the supplier from the store.
		k.RemoveSupplier(ctx, supplierOperatorAddress.String())
		logger.Info(fmt.Sprintf("Successfully removed the supplier: %+v", supplier))

		unbondingReason := suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_VOLUNTARY
		supplierParams := k.GetParamsAtHeight(ctx, int64(supplier.UnstakeSessionEndHeight))
		if supplier.GetStake().Amount.LT(supplierParams.MinStake.Amount) {
			unbondingReason = suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE
		}

		// Emit an event which signals that the supplier has successfully unbonded.
		sessionEndHeight := sharedtypes.GetSessionEndHeight(sharedParamsUpdates, currentHeight)
		unbondingEndEvent := &suppliertypes.EventSupplierUnbondingEnd{
			Supplier:           &supplier,
			Reason:             unbondingReason,
			SessionEndHeight:   sessionEndHeight,
			UnbondingEndHeight: unbondingEndHeight,
		}
		if eventErr := sdkCtx.EventManager().EmitTypedEvent(unbondingEndEvent); eventErr != nil {
			logger.Error(fmt.Sprintf("failed to emit event: %+v; %s", unbondingEndEvent, eventErr))
			return numUnbondedSuppliers, eventErr
		}

		numUnbondedSuppliers += 1
	}

	return numUnbondedSuppliers, nil
}
