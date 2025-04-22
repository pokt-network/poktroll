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
	sharedParams := k.sharedKeeper.GetParams(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Only process unbonding suppliers at the end of the session.
	if sharedtypes.IsSessionEndHeight(&sharedParams, currentHeight) {
		return numUnbondedSuppliers, nil
	}

	logger := k.Logger().With("method", "UnbondSupplier")

	// Iterate over all suppliers and unbond suppliers that have finished the unbonding period.
	// TODO_POST_MAINNET(@red-0ne): Use an index to iterate over suppliers that have initiated the
	// unbonding action instead of iterating over all suppliers.
	allSuppliersIterator := k.GetAllSuppliersIterator(ctx)
	defer allSuppliersIterator.Close()
	for ; allSuppliersIterator.Valid(); allSuppliersIterator.Next() {
		supplier, err := allSuppliersIterator.Value()
		if err != nil {
			logger.Error(fmt.Sprintf("could not get supplier from iterator: %v", err))
			return numUnbondedSuppliers, err
		}

		// Ignore suppliers that have not initiated the unbonding action.
		if !supplier.IsUnbonding() {
			continue
		}

		unbondingEndHeight := sharedtypes.GetSupplierUnbondingEndHeight(&sharedParams, &supplier)

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

		// Remove the supplier from the store.
		k.RemoveSupplier(ctx, supplierOperatorAddress.String())
		logger.Info(fmt.Sprintf("Successfully removed the supplier: %+v", supplier))

		unbondingReason := suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_VOLUNTARY
		if supplier.GetStake().Amount.LT(k.GetParams(ctx).MinStake.Amount) {
			unbondingReason = suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE
		}

		// Emit an event which signals that the supplier has sucessfully unbonded.
		sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
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
