package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// DeactivateSupplierService schedules deactivation, at the next session start, of
// the supplier's service configs for serviceId that are still in effect then, and
// emits EventSupplierServiceNotAllowed. It returns false, changing nothing, if the
// supplier is unbonding or has no such config. It never panics or returns an error.
//
// Same caller contract as EnforceServiceCap: a caller holding a hydrated copy of
// this supplier must write it back only with SetDehydratedSupplier.
func (k Keeper) DeactivateSupplierService(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	serviceId string,
) (deactivated bool) {
	if supplier.IsUnbonding() {
		return false
	}

	nextSessionStartHeight := k.nextSessionStartHeight(ctx)
	for _, config := range k.countedServiceConfigs(ctx, supplier.OperatorAddress, nextSessionStartHeight) {
		if config.GetService().GetServiceId() != serviceId {
			continue
		}
		k.rescheduleServiceConfigDeactivation(ctx, config, nextSessionStartHeight)
		deactivated = true
	}
	if !deactivated {
		return false
	}

	event := &suppliertypes.EventSupplierServiceNotAllowed{
		OperatorAddress: supplier.OperatorAddress,
		OwnerAddress:    supplier.OwnerAddress,
		ServiceId:       serviceId,
		EffectiveHeight: nextSessionStartHeight,
	}
	if err := cosmostypes.UnwrapSDKContext(ctx).EventManager().EmitTypedEvent(event); err != nil {
		k.Logger().Error(fmt.Sprintf("failed to emit EventSupplierServiceNotAllowed: %+v; %s", event, err))
	}
	return true
}

// enforceSupplierAllowlists deactivates, at the next session start, every service
// of the supplier whose supplier allowlist lists neither its owner nor its
// operator. It returns the deactivated service IDs in store key order.
func (k Keeper) enforceSupplierAllowlists(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
) (deactivatedServiceIds []string) {
	for _, config := range k.countedServiceConfigs(ctx, supplier.OperatorAddress, k.nextSessionStartHeight(ctx)) {
		serviceId := config.GetService().GetServiceId()
		if k.serviceKeeper.IsSupplierAllowed(ctx, serviceId, supplier.OwnerAddress, supplier.OperatorAddress) {
			continue
		}
		// A second config for an already deactivated service returns false.
		if k.DeactivateSupplierService(ctx, supplier, serviceId) {
			deactivatedServiceIds = append(deactivatedServiceIds, serviceId)
		}
	}
	return deactivatedServiceIds
}
