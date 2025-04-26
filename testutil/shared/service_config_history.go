package shared

import (
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	// NoDeactivationHeight represents that a service configuration has no deactivation
	// height and is considered active indefinitely.
	NoDeactivationHeight = iota // 0
)

// CreateServiceConfigUpdateHistoryFromServiceConfigs creates a list of ServiceConfigUpdate
// objects from a list of SupplierServiceConfig objects.
// - This utility function is used in tests to construct the service configuration history for a supplier.
// - Returns a slice of ServiceConfigUpdate objects that can be assigned to a Supplier's ServiceConfigHistory field.
func CreateServiceConfigUpdateHistoryFromServiceConfigs(
	operatorAddress string,
	services []*sharedtypes.SupplierServiceConfig,
	activationHeight int64,
	deactivationHeight int64,
) []*sharedtypes.ServiceConfigUpdate {
	serviceConfigHistory := make([]*sharedtypes.ServiceConfigUpdate, 0, len(services))
	for _, service := range services {
		serviceConfigUpdate := CreateServiceConfigUpdateFromServiceConfig(
			operatorAddress,
			service,
			activationHeight,
			deactivationHeight,
		)

		serviceConfigHistory = append(serviceConfigHistory, serviceConfigUpdate)
	}
	return serviceConfigHistory
}

// CreateServiceConfigUpdateFromServiceConfig creates a single ServiceConfigUpdate
// object from a SupplierServiceConfig object.
// - This utility function creates a single service configuration update.
// - Returns a single ServiceConfigUpdate object.
func CreateServiceConfigUpdateFromServiceConfig(
	operatorAddress string,
	service *sharedtypes.SupplierServiceConfig,
	activationHeight int64,
	deactivationHeight int64,
) *sharedtypes.ServiceConfigUpdate {
	return &sharedtypes.ServiceConfigUpdate{
		OperatorAddress:    operatorAddress,
		Service:            service,
		ActivationHeight:   activationHeight,
		DeactivationHeight: deactivationHeight,
	}
}
