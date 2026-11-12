package keeper

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// MigrateNonCanonicalSupplierAddresses handles supplier addresses stored in a
// non-canonical bech32 encoding (accepted before MsgStakeSupplier required the
// canonical form):
//   - a supplier stored under a non-canonical operator address is removed with
//     its indexes and its stake is NOT refunded: it stays in the supplier module
//     pool (EventSupplierStakeStuckInModulePool + EventSupplierUnbondingEnd);
//   - a non-canonical owner address is rewritten to its canonical form, so the
//     owner can still sign for the supplier now that signers must be canonical.
//
// Every key read or deleted is the stored string, so a supplier stored under
// the canonical encoding of the same account is never removed.
func (k Keeper) MigrateNonCanonicalSupplierAddresses(ctx context.Context) (numRemoved, numOwnersCanonicalized int) {
	logger := k.Logger().With("method", "MigrateNonCanonicalSupplierAddresses")
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)

	// Collect first: the loops below write to the store being iterated.
	var suppliers []sharedtypes.Supplier
	iterator := storetypes.KVStorePrefixIterator(k.getSupplierStore(ctx), []byte{})
	for ; iterator.Valid(); iterator.Next() {
		var supplier sharedtypes.Supplier
		k.cdc.MustUnmarshal(iterator.Value(), &supplier)
		suppliers = append(suppliers, supplier)
	}
	iterator.Close()

	for _, supplier := range suppliers {
		if sharedtypes.ValidateCanonicalAddress(supplier.OperatorAddress) != nil {
			k.RemoveSupplier(ctx, supplier.OperatorAddress)
			numRemoved++
			logger.Info(fmt.Sprintf("removed supplier with non-canonical operator address %s; stake %s retained in module pool", supplier.OperatorAddress, supplier.Stake))

			events := []cosmostypes.Msg{
				&suppliertypes.EventSupplierStakeStuckInModulePool{
					OperatorAddress:  supplier.OperatorAddress,
					OwnerAddress:     supplier.OwnerAddress,
					StuckCoin:        supplier.Stake,
					Reason:           "non-canonical operator address",
					SessionEndHeight: sessionEndHeight,
				},
				&suppliertypes.EventSupplierUnbondingEnd{
					Supplier:           &supplier,
					Reason:             suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_MIGRATION,
					SessionEndHeight:   sessionEndHeight,
					UnbondingEndHeight: currentHeight,
				},
			}
			for _, event := range events {
				if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
					logger.Error(fmt.Sprintf("failed to emit %T for supplier %s: %s", event, supplier.OperatorAddress, err))
				}
			}
			continue
		}

		ownerAddress, err := cosmostypes.AccAddressFromBech32(supplier.OwnerAddress)
		if err != nil || ownerAddress.String() == supplier.OwnerAddress {
			continue
		}
		logger.Info(fmt.Sprintf("canonicalized owner address %s of supplier %s", supplier.OwnerAddress, supplier.OperatorAddress))
		supplier.OwnerAddress = ownerAddress.String()
		k.SetDehydratedSupplier(ctx, supplier)
		numOwnersCanonicalized++
	}

	return numRemoved, numOwnersCanonicalized
}
