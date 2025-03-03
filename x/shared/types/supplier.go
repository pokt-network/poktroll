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

// IsActive returns whether the supplier is allowed to serve requests for the
// given serviceId and query height.
// A supplier is active for a given service starting from the session following
// the one during which the supplier staked for that service.
// A supplier that has submitted an unstake message is active until the end of
// the session containing the height at which unstake message was submitted.
func (s *Supplier) IsActive(queryHeight uint64, serviceId string) bool {
	var activeServices []*SupplierServiceConfig
	for _, supplierServiceUpdate := range s.ServicesUpdateHistory {
		activeServices = supplierServiceUpdate.Services
		if supplierServiceUpdate.UpdateHeight > queryHeight {
			break
		}
	}

	if activeServices == nil {
		return false
	}

	containsServiceId := func(config *SupplierServiceConfig) bool {
		return config.ServiceId == serviceId
	}

	return slices.ContainsFunc(activeServices, containsServiceId)
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
