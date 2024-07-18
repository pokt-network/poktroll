package shared

import (
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func GetSupplierUnbondingHeight(
	sharedParams *sharedtypes.Params,
	supplier *sharedtypes.Supplier,
) int64 {
	return GetProofWindowCloseHeight(sharedParams, int64(supplier.UnstakeSessionEndHeight))
}
