package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// BeginSupplierUnbonding starts unbonding supplier at the end of the current
// session and persists it, service config indexes included. It sets
// supplier.UnstakeSessionEndHeight and returns it.
//
// Every service config still active at the next session start is scheduled to
// deactivate there, so the supplier keeps serving the current session and the
// session's supplier set never changes mid-session. A config that is already
// deactivated keeps its height: moving it later would reactivate it for past
// sessions and change which suppliers they contain.
//
// supplier MAY be dehydrated (e.g. from the settlement context): the service
// config history is always reloaded from the index, and only the configs whose
// deactivation height changes are rewritten.
//
// sharedParams MUST be the live shared params: the unstake session end and the
// next session start are projected from the current block.
func (k Keeper) BeginSupplierUnbonding(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	sharedParams *sharedtypes.Params,
) (unstakeSessionEndHeight int64) {
	currentHeight := cosmostypes.UnwrapSDKContext(ctx).BlockHeight()
	unstakeSessionEndHeight = sharedtypes.GetSessionEndHeight(sharedParams, currentHeight)
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(sharedParams, currentHeight)

	supplier.UnstakeSessionEndHeight = uint64(unstakeSessionEndHeight)

	for _, serviceConfig := range k.getSupplierServiceConfigUpdates(ctx, supplier.OperatorAddress, "") {
		if suppliertypes.IsCountedServiceConfig(serviceConfig, nextSessionStartHeight) {
			k.rescheduleServiceConfigDeactivation(ctx, serviceConfig, nextSessionStartHeight)
		}
	}

	k.indexSupplierUnstakingHeight(ctx, *supplier)
	k.SetDehydratedSupplier(ctx, *supplier)

	return unstakeSessionEndHeight
}

// GetSupplierUnbondingEndHeight returns the height at which EndBlockerUnbondSuppliers
// unbonds the given unbonding supplier. It uses the shared params effective at the
// supplier's unstake session end, NOT the live params: a later num_blocks_per_session
// decrease would otherwise shrink the unbonding window and release the supplier's
// stake before its in-flight claims settle (#543, F1).
func (k Keeper) GetSupplierUnbondingEndHeight(ctx context.Context, supplier *sharedtypes.Supplier) int64 {
	unstakeParams := k.sharedKeeper.GetParamsAtHeight(ctx, int64(supplier.GetUnstakeSessionEndHeight()))
	return sharedtypes.GetSupplierUnbondingEndHeight(&unstakeParams, supplier)
}

// RepairUnindexedUnbondingSuppliers runs BeginSupplierUnbonding again for every
// supplier that is unbonding but missing from the unstaking index, and returns
// their operator addresses in store order.
//
// Before BeginSupplierUnbonding existed, settlement force-unstaked a supplier
// below min stake without indexing it: it stayed selectable in sessions, was
// never unbonded, and MsgUnstakeSupplier rejected it as already unbonding.
// Settlement now skips such a supplier (it is already unbonding), so it needs
// this one-shot repair from an upgrade handler.
//
// The unstake session end height is reset to the current session end, restarting
// the unbonding period. A stuck supplier is still in the current session, so
// keeping its old (past) height would let EndBlockerUnbondSuppliers remove it
// mid-session and orphan the claims of the sessions it serves.
//
// Suppliers already in the index (voluntary unstakes, Morse claims) are left
// untouched. Each repaired supplier gets a new EventSupplierUnbondingBegin, since
// its unbonding restarts with new heights.
func (k Keeper) RepairUnindexedUnbondingSuppliers(ctx context.Context) (repaired []string) {
	indexed := make(map[string]struct{})
	unstakingIterator := k.GetAllUnstakingSuppliersIterator(ctx)
	for ; unstakingIterator.Valid(); unstakingIterator.Next() {
		indexed[string(unstakingIterator.Value())] = struct{}{}
	}
	unstakingIterator.Close()

	sharedParams := k.sharedKeeper.GetParams(ctx)

	// GetAllSuppliers returns a slice, so the writes below never run while a
	// supplier store iterator is open.
	for _, supplier := range k.GetAllSuppliers(ctx) {
		if !supplier.IsUnbonding() {
			continue
		}
		if _, ok := indexed[supplier.OperatorAddress]; ok {
			continue
		}
		unstakeSessionEndHeight := k.BeginSupplierUnbonding(ctx, &supplier, &sharedParams)
		repaired = append(repaired, supplier.OperatorAddress)

		// The event carries the dehydrated supplier, like the voluntary unbonding
		// event: the hydrated copy predates the deactivations just stored.
		supplier.Services = nil
		supplier.ServiceConfigHistory = nil
		event := &suppliertypes.EventSupplierUnbondingBegin{
			Supplier:           &supplier,
			Reason:             suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE,
			SessionEndHeight:   unstakeSessionEndHeight,
			UnbondingEndHeight: k.GetSupplierUnbondingEndHeight(ctx, &supplier),
		}
		if err := cosmostypes.UnwrapSDKContext(ctx).EventManager().EmitTypedEvent(event); err != nil {
			k.Logger().Error(fmt.Sprintf("failed to emit EventSupplierUnbondingBegin: %+v; %s", event, err))
		}
	}
	return repaired
}

// EndBlockerUnbondSuppliers unbonds suppliers whose unbonding period has elapsed.
func (k Keeper) EndBlockerUnbondSuppliers(ctx context.Context) (numUnbondedSuppliers uint64, err error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Only process unbonding suppliers at the end of the session.
	if !sharedtypes.IsSessionEndHeight(&sharedParams, currentHeight) {
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

		unbondingEndHeight := k.GetSupplierUnbondingEndHeight(ctx, &supplier)

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
			// If the transfer fails (e.g., a legacy module-account owner — new
			// occurrences are blocked by the stake-time module-account-owner
			// check, but pre-v0.1.34 state can still trip this), log the error,
			// emit EventSupplierStakeStuckInModulePool for indexer/governance
			// visibility, and continue.
			//
			// Why not halt the chain: pre-existing legacy state must not be
			// allowed to brick the EndBlocker. Coins remain in the supplier
			// module pool; the event surfaces them for a governance reclaim
			// path. Removing the supplier from state keeps the unbonding queue
			// making progress and prevents an infinite-retry on the same dead
			// entry every session-end.
			if err = k.bankKeeper.SendCoinsFromModuleToAccount(
				ctx, suppliertypes.ModuleName, ownerAddress, []cosmostypes.Coin{*supplier.Stake},
			); err != nil {
				logger.Error(fmt.Sprintf(
					"could not send %s coins from module %s to account %s due to %s; supplier will be removed and coins will remain in module pool (see EventSupplierStakeStuckInModulePool)",
					supplier.Stake.String(), suppliertypes.ModuleName, ownerAddress, err,
				))

				stuckSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
				stuckEvent := &suppliertypes.EventSupplierStakeStuckInModulePool{
					OperatorAddress:  supplier.OperatorAddress,
					OwnerAddress:     supplier.OwnerAddress,
					StuckCoin:        supplier.Stake,
					Reason:           err.Error(),
					SessionEndHeight: stuckSessionEndHeight,
				}
				if emitErr := sdkCtx.EventManager().EmitTypedEvent(stuckEvent); emitErr != nil {
					logger.Error(fmt.Sprintf("failed to emit EventSupplierStakeStuckInModulePool: %+v; %s", stuckEvent, emitErr))
				}
			}
		}

		// Remove the supplier from the store.
		k.RemoveSupplier(ctx, supplierOperatorAddress.String())
		logger.Info(fmt.Sprintf("Successfully removed the supplier: %+v", supplier))

		// Defensive: GetParams returns a zero-value Params{} (nil MinStake) if params
		// were never written. Fall back to DefaultMinStake to avoid a nil-deref that
		// would halt the chain at the EndBlocker.
		minStake := suppliertypes.DefaultMinStake
		if supMinStake := k.GetParams(ctx).MinStake; supMinStake != nil {
			minStake = *supMinStake
		}
		unbondingReason := suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_VOLUNTARY
		if supplier.GetStake().Amount.LT(minStake.Amount) {
			unbondingReason = suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE
		}

		// Emit an event which signals that the supplier has successfully unbonded.
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
