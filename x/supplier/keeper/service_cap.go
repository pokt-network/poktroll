package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// EnforceServiceCap schedules deactivation, at the next session start, of the
// supplier's services beyond what its stake covers, keeping services in
// service_priority order. It never unbonds a supplier: a stake below min_stake
// is left to the existing unbonding path. It never panics or returns an error.
//
// supplier must be the caller's live copy (in settlement, the settlement
// context's cached pointer), since callers may write it again afterwards.
func (k Keeper) EnforceServiceCap(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	reason suppliertypes.ServiceCapReason,
) (deactivatedServiceIds []string) {
	if supplier.IsUnbonding() || supplier.Stake == nil {
		return nil
	}

	allowed, limited := suppliertypes.AllowedServices(*supplier.Stake, k.GetParams(ctx))
	if !limited || allowed == 0 {
		return nil
	}

	sharedParams := k.sharedKeeper.GetParams(ctx)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, sdkCtx.BlockHeight())

	serviceConfigHistory := k.getSupplierServiceConfigUpdates(ctx, supplier.OperatorAddress, "")
	counted := make([]*sharedtypes.ServiceConfigUpdate, 0, len(serviceConfigHistory))
	for _, config := range serviceConfigHistory {
		if suppliertypes.IsCountedServiceConfig(config, nextSessionStartHeight) {
			counted = append(counted, config)
		}
	}
	if uint64(len(counted)) <= allowed {
		return nil
	}

	// Ranked holds the same pointers as serviceConfigHistory.
	ranked := suppliertypes.RankServiceConfigs(counted, supplier.ServicePriority)
	for _, config := range ranked[allowed:] {
		config.DeactivationHeight = nextSessionStartHeight
		deactivatedServiceIds = append(deactivatedServiceIds, config.Service.ServiceId)
	}

	supplierToIndex := *supplier
	supplierToIndex.ServiceConfigHistory = serviceConfigHistory
	k.SetAndIndexDehydratedSupplier(ctx, supplierToIndex)

	event := &suppliertypes.EventSupplierServicesCapped{
		OperatorAddress:       supplier.OperatorAddress,
		OwnerAddress:          supplier.OwnerAddress,
		Stake:                 supplier.Stake,
		AllowedServices:       allowed,
		DeactivatedServiceIds: deactivatedServiceIds,
		Reason:                reason,
		EffectiveHeight:       nextSessionStartHeight,
	}
	if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
		k.Logger().Error(fmt.Sprintf("failed to emit EventSupplierServicesCapped: %+v; %s", event, err))
	}

	return deactivatedServiceIds
}
