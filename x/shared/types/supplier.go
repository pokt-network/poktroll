package types

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
//
// This method examines the supplier's service configuration history to determine
// if they have an active configuration for the requested service ID at the
// specified block height. A supplier is considered "active" for a service when:
//  1. They have a ServiceConfigUpdate for this service ID
//  2. That configuration is active at the given block height
//     (activation height <= queryHeight < deactivation height)
func (s *Supplier) IsActive(queryHeight int64, serviceId string) bool {
	// Examine each service configuration update in the history
	for _, serviceUpdate := range s.ServiceConfigHistory {
		// Skip configurations for other services
		if serviceUpdate.Service.ServiceId != serviceId {
			continue
		}

		if serviceUpdate.IsActive(queryHeight) {
			return true
		}
	}

	// No active configuration was found for this service at the given height
	return false
}

// GetActiveServiceConfigs returns a list of all service configurations that are active
// at the specified block height.
//
// This method examines the supplier's service configuration history to collect
// all service configurations that:
//  1. Have an activation height less than or equal to the query height
//  2. Either have no deactivation height (0) or a deactivation height greater than the query height
//
// The returned configurations represent all services the supplier is authorized to provide
// at the given block height, with their corresponding endpoints and revenue share settings.
func (s *Supplier) GetActiveServiceConfigs(
	queryHeight int64,
) []*SupplierServiceConfig {
	activeServiceConfigs := make([]*SupplierServiceConfig, 0)
	for _, serviceConfigUpdate := range s.ServiceConfigHistory {
		if serviceConfigUpdate.IsActive(queryHeight) {
			activeServiceConfigs = append(activeServiceConfigs, serviceConfigUpdate.Service)
		}
	}
	return activeServiceConfigs
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
//
// This calculates the absolute block height at which the supplier's unbonding period
// completes by adding the configured unbonding period (in sessions) to the supplier's
// unstake session end height.
func GetSupplierUnbondingEndHeight(
	sharedParams *Params,
	supplier *Supplier,
) int64 {
	// Calculate the number of blocks in the unbonding period
	supplierUnbondingPeriodBlocks := sharedParams.GetSupplierUnbondingPeriodSessions() * sharedParams.GetNumBlocksPerSession()

	// Add the unbonding period to the session end height to get the final unbonding height
	return int64(supplier.GetUnstakeSessionEndHeight() + supplierUnbondingPeriodBlocks)
}
