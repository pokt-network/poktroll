package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func (k Keeper) EndBlockerPruneSupplierServicesUpdateHistory(ctx context.Context) error {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sessionEndToProofWindowCloseNumBlocks := sharedtypes.GetSessionEndToProofWindowCloseBlocks(&sharedParams)

	logger := k.Logger().With("method", "PruneSupplierServicesUpdateHistory")

	for _, supplier := range k.GetAllSuppliers(ctx) {
		previousSupplierServicesUpdateHistoryLen := len(supplier.ServicesUpdateHistory)
		updatedSupplierServicesHistory := make([]*sharedtypes.ServicesUpdate, 0)
		for _, servicesUpdate := range supplier.ServicesUpdateHistory {
			updateSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, int64(servicesUpdate.UpdateHeight))
			updateSessionSettlementHeight := sessionEndToProofWindowCloseNumBlocks + updateSessionEndHeight
			if currentHeight <= updateSessionSettlementHeight {
				updatedSupplierServicesHistory = append(updatedSupplierServicesHistory, servicesUpdate)
			}
		}

		if len(updatedSupplierServicesHistory) == previousSupplierServicesUpdateHistoryLen {
			continue
		}

		if len(updatedSupplierServicesHistory) == 0 {
			updatedSupplierServicesHistory = supplier.ServicesUpdateHistory[:1]
		}

		supplier.ServicesUpdateHistory = updatedSupplierServicesHistory

		k.SetSupplier(ctx, supplier)
		logger.Info(fmt.Sprintf(
			"Pruned %d services update history entries for supplier %s",
			previousSupplierServicesUpdateHistoryLen-len(updatedSupplierServicesHistory),
			supplier.OperatorAddress,
		))
	}

	return nil
}
