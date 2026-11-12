package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// SupplierOwnerRecovery names a supplier, by operator address, whose owner address
// is rewritten to NewOwnerAddress by RecoverSupplierOwners.
type SupplierOwnerRecovery struct {
	OperatorAddress string
	NewOwnerAddress string
}

// RecoverSupplierOwners rewrites the owner address of each listed supplier, keyed by
// operator address, and optionally cancels its unbonding. It is an upgrade-handler
// remedy for suppliers whose owner address was changed without the owner's consent
// and that were then unstaked: the unbonding refund follows the stored owner. The
// stored owner is NOT checked, because it can change again before the upgrade; the
// operator address is the store key and cannot change.
//
// When cancelUnbonding is true an unbonding supplier stays staked under the new owner
// (EventSupplierUnbondingCanceled is emitted); its services stay as stored, which is
// empty when the unstake cleared them. When false every listed supplier ends up
// unbonding under the new owner: an unbonding found in progress is kept as is, and a
// supplier found staked begins a voluntary unbonding (EventSupplierUnbondingBegin), so
// the refund path is the same for the whole list. In both cases the below-min-stake
// marker is cleared, so from here on only the new owner can cancel the unbonding.
//
// Entries are skipped, never fatal: a missing operator (already unbonded), or a new
// owner that is not a canonical bech32 address or cannot receive funds (the refund
// would get stuck in the module pool). The caller passes a deterministic slice.
func (k Keeper) RecoverSupplierOwners(
	ctx context.Context,
	recoveries []SupplierOwnerRecovery,
	cancelUnbonding bool,
) (numRecovered, numSkipped int) {
	logger := k.Logger().With("method", "RecoverSupplierOwners")
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)

	for _, recovery := range recoveries {
		if err := sharedtypes.ValidateCanonicalAddress(recovery.NewOwnerAddress); err != nil {
			logger.Error(fmt.Sprintf("skipping supplier %s: invalid new owner address: %s", recovery.OperatorAddress, err))
			numSkipped++
			continue
		}
		newOwnerAddr := cosmostypes.MustAccAddressFromBech32(recovery.NewOwnerAddress)
		_, isModuleAccount := k.accountKeeper.GetAccount(ctx, newOwnerAddr).(cosmostypes.ModuleAccountI)
		if isModuleAccount || k.bankKeeper.BlockedAddr(newOwnerAddr) {
			logger.Error(fmt.Sprintf("skipping supplier %s: new owner address %s cannot receive funds", recovery.OperatorAddress, recovery.NewOwnerAddress))
			numSkipped++
			continue
		}

		supplier, found := k.GetDehydratedSupplier(ctx, recovery.OperatorAddress)
		if !found {
			logger.Info(fmt.Sprintf("skipping supplier %s: not found", recovery.OperatorAddress))
			numSkipped++
			continue
		}

		previousOwner := supplier.OwnerAddress
		supplier.OwnerAddress = recovery.NewOwnerAddress

		switch {
		case cancelUnbonding && supplier.IsUnbonding():
			supplier.UnstakeSessionEndHeight = sharedtypes.SupplierNotUnstaking
			k.removeBelowMinStakeUnbonding(ctx, supplier.OperatorAddress)
			k.indexSupplierUnstakingHeight(ctx, supplier)
			k.SetDehydratedSupplier(ctx, supplier)
			event := &suppliertypes.EventSupplierUnbondingCanceled{
				Supplier:         &supplier,
				Height:           currentHeight,
				SessionEndHeight: sessionEndHeight,
			}
			if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
				logger.Error(fmt.Sprintf("failed to emit EventSupplierUnbondingCanceled for supplier %s: %s", supplier.OperatorAddress, err))
			}
		case !cancelUnbonding && !supplier.IsUnbonding():
			// Same steps as MsgUnstakeSupplier: the services deactivate at the next
			// session start and the stake is refunded at the unbonding end.
			k.BeginSupplierUnbonding(ctx, &supplier, &sharedParams, suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_VOLUNTARY)
			event := &suppliertypes.EventSupplierUnbondingBegin{
				Supplier:           &supplier,
				Reason:             suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_VOLUNTARY,
				SessionEndHeight:   int64(supplier.GetUnstakeSessionEndHeight()),
				UnbondingEndHeight: k.GetSupplierUnbondingEndHeight(ctx, &supplier),
			}
			if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
				logger.Error(fmt.Sprintf("failed to emit EventSupplierUnbondingBegin for supplier %s: %s", supplier.OperatorAddress, err))
			}
		default:
			// Unbonding kept (or already staked with cancelUnbonding): same write path as a
			// stake-only MsgStakeSupplier, the service config history and its indexes are
			// left untouched.
			k.removeBelowMinStakeUnbonding(ctx, supplier.OperatorAddress)
			k.indexSupplierUnstakingHeight(ctx, supplier)
			k.SetDehydratedSupplier(ctx, supplier)
		}

		numRecovered++
		logger.Info(fmt.Sprintf("recovered supplier %s: owner %s -> %s, unbonding=%t",
			supplier.OperatorAddress, previousOwner, supplier.OwnerAddress, supplier.IsUnbonding()))
	}

	return numRecovered, numSkipped
}
