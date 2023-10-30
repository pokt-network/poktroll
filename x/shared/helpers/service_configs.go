package helpers

import (
	"fmt"

	sharedtypes "pocket/x/shared/types"
)

// AreValidAppServiceConfigs returns an error if any of the application service configs are invalid
func AreValidAppServiceConfigs(services []*sharedtypes.ApplicationServiceConfig) error {
	if len(services) == 0 {
		return fmt.Errorf("no services configs provided for application: %v", services)
	}
	for _, serviceConfig := range services {
		if serviceConfig == nil {
			return fmt.Errorf("serviceConfig cannot be nil: %v", services)
		}
		if serviceConfig.ServiceId == nil {
			return fmt.Errorf("serviceId cannot be nil: %v", serviceConfig)
		}
		if serviceConfig.ServiceId.Id == "" {
			return fmt.Errorf("serviceId.Id cannot be empty: %v", serviceConfig)
		}
		if !IsValidServiceId(serviceConfig.ServiceId.Id) {
			return fmt.Errorf("invalid serviceId.Id: %v", serviceConfig)
		}
		if !IsValidServiceName(serviceConfig.ServiceId.Name) {
			return fmt.Errorf("invalid serviceId.Name: %v", serviceConfig)
		}
	}
	return nil
}

// AreValidSupplierServiceConfigs returns an error if any of the supplier service configs are invalid
func AreValidSupplierServiceConfigs(services []*sharedtypes.SupplierServiceConfig) error {
	if len(services) == 0 {
		return fmt.Errorf("no services configs provided for supplier: %v", services)
	}
	for _, serviceConfig := range services {
		if serviceConfig == nil {
			return fmt.Errorf("serviceConfig cannot be nil: %v", services)
		}

		// Check the ServiceId
		if serviceConfig.ServiceId == nil {
			return fmt.Errorf("serviceId cannot be nil: %v", serviceConfig)
		}
		if serviceConfig.ServiceId.Id == "" {
			return fmt.Errorf("serviceId.Id cannot be empty: %v", serviceConfig)
		}
		if !IsValidServiceId(serviceConfig.ServiceId.Id) {
			return fmt.Errorf("invalid serviceId.Id: %v", serviceConfig)
		}
		if !IsValidServiceName(serviceConfig.ServiceId.Name) {
			return fmt.Errorf("invalid serviceId.Name: %v", serviceConfig)
		}

		// Check the Endpoints
		if serviceConfig.Endpoints == nil {
			return fmt.Errorf("endpoints cannot be nil: %v", serviceConfig)
		}
		if len(serviceConfig.Endpoints) == 0 {
			return fmt.Errorf("endpoints must have at least one entry: %v", serviceConfig)
		}

		// Check each endpoint
		for _, endpoint := range serviceConfig.Endpoints {
			if endpoint == nil {
				return fmt.Errorf("endpoint cannot be nil: %v", serviceConfig)
			}
			if endpoint.Url == "" {
				return fmt.Errorf("endpoint.Url cannot be empty: %v", serviceConfig)
			}
			if !IsValidEndpointUrl(endpoint.Url) {
				return fmt.Errorf("invalid endpoint.Url: %v", serviceConfig)
			}
			// TODO_TECHDEBT: Verify endpoint.ConfigOptions and endpoint.RpcType
		}
	}
	return nil
}
