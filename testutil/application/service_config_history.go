package application

import (
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// CreateServiceConfigUpdateHistoryFromServiceConfigs creates a list of
// ApplicationServiceConfigUpdate objects from a list of ApplicationServiceConfig
// objects.
//   - This utility function is used in tests to construct the service configuration
//     history for an application (the app-side analogue of the supplier helper in
//     testutil/shared/service_config_history.go).
//   - Returns a slice that can be assigned to an Application's ServiceConfigHistory field.
func CreateServiceConfigUpdateHistoryFromServiceConfigs(
	applicationAddress string,
	services []*sharedtypes.ApplicationServiceConfig,
	activationHeight int64,
	deactivationHeight int64,
) []*apptypes.ApplicationServiceConfigUpdate {
	serviceConfigHistory := make([]*apptypes.ApplicationServiceConfigUpdate, 0, len(services))
	for _, service := range services {
		serviceConfigHistory = append(serviceConfigHistory, CreateServiceConfigUpdateFromServiceConfig(
			applicationAddress,
			service,
			activationHeight,
			deactivationHeight,
		))
	}
	return serviceConfigHistory
}

// CreateServiceConfigUpdateFromServiceConfig creates a single
// ApplicationServiceConfigUpdate object from an ApplicationServiceConfig object.
func CreateServiceConfigUpdateFromServiceConfig(
	applicationAddress string,
	service *sharedtypes.ApplicationServiceConfig,
	activationHeight int64,
	deactivationHeight int64,
) *apptypes.ApplicationServiceConfigUpdate {
	return &apptypes.ApplicationServiceConfigUpdate{
		ApplicationAddress: applicationAddress,
		Service:            service,
		ActivationHeight:   activationHeight,
		DeactivationHeight: deactivationHeight,
	}
}
