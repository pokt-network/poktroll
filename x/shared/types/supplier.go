package types

import "slices"

// SupplierNotUnstaking is the value of `unstake_session_end_height` if the
// supplier is not actively in the unbonding period.
const SupplierNotUnstaking uint64 = iota

// IsUnbonding returns true if the supplier is actively unbonding.
// It determines if the supplier has submitted an unstake message, in which case
// the supplier has its UnstakeSessionEndHeight set.
func (s *Supplier) IsUnbonding() bool {
	return s.UnstakeSessionEndHeight != SupplierNotUnstaking
}

// IsActive checks if the supplier is authorized to serve requests for a specific service
// at the given block height.
func (s *Supplier) IsActive(queryHeight uint64, serviceId string) bool {
	// Keep track of the most recent set of service configs that were active
	// before the query height.
	var servicesAtHeight []*SupplierServiceConfig

	// Iterate through the service config history chronologically
	for _, serviceUpdate := range s.ServiceConfigHistory {
		// If this update takes effect after our query height, stop looking.
		// We want the last update that was active before the query height.
		if serviceUpdate.EffectiveBlockHeight > queryHeight {
			break
		}
		// Keep updating our services list as we move forward in time.
		servicesAtHeight = serviceUpdate.Services
	}

	// If we found no service configurations active at this height, supplier is not active.
	if servicesAtHeight == nil {
		return false
	}

	// Define a helper function to check if a service config matches our target service ID.
	matchesServiceIdFn := func(config *SupplierServiceConfig) bool {
		return config.ServiceId == serviceId
	}

	// Check if any of the active services match our target service ID
	return slices.ContainsFunc(servicesAtHeight, matchesServiceIdFn)
}

// HasOwner returns whether the given address is the supplier's owner address.
func (s *Supplier) HasOwner(address string) bool {
	return s.OwnerAddress == address
}

// HasOperator returns whether the given address is the supplier's operator address.
func (s *Supplier) HasOperator(address string) bool {
	return s.OperatorAddress == address
}

// GetSupplierUnbondingEndHeight returns the session end height at which the given
// supplier finishes unbonding.
func GetSupplierUnbondingEndHeight(
	sharedParams *Params,
	supplier *Supplier,
) int64 {
	supplierUnbondingPeriodBlocks := sharedParams.GetSupplierUnbondingPeriodSessions() * sharedParams.GetNumBlocksPerSession()

	return int64(supplier.GetUnstakeSessionEndHeight() + supplierUnbondingPeriodBlocks)
}
