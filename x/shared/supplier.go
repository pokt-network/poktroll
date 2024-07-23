package shared

import (
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// GetProofWindowCloseHeight returns the height at which the given supplier finishes unbonding.
func GetSupplierUnbondingHeight(
	sharedParams *sharedtypes.Params,
	supplier *sharedtypes.Supplier,
) int64 {
	// TODO_UPNEXT(red-0ne): Add a governance parameter called `supplier_unbonding_period`
	// equal to the number of blocks required to unbond. The value should enforce
	// (when being updated) to be after proof window close height and should still
	// round to the end of the nearest session.
	return GetProofWindowCloseHeight(sharedParams, int64(supplier.UnstakeSessionEndHeight))
}
