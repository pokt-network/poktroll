package shared

import (
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// GetSupplierUnbondingHeight returns the session end height at which the given
// supplier finishes unbonding.
func GetSupplierUnbondingHeight(
	sharedParams *sharedtypes.Params,
	supplier *sharedtypes.Supplier,
) int64 {
	supplierUnbondingPeriodSessions := sharedParams.SupplierUnbondingPeriodSessions * sharedParams.NumBlocksPerSession

	return int64(supplier.UnstakeSessionEndHeight + supplierUnbondingPeriodSessions)
}
