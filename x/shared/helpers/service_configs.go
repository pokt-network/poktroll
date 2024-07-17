package helpers

import (
	"fmt"

	sharedtypes "github.com/pokt-network/poktroll/proto/types/shared"
)

// ValidateAppServiceConfigs returns an error if any of the application service configs are invalid
func ValidateAppServiceConfigs(services []*sharedtypes.ApplicationServiceConfig) error {
	if len(services) == 0 {
		return fmt.Errorf("no services configs provided for application: %v", services)
	}
	for _, serviceConfig := range services {
		if serviceConfig == nil {
			return fmt.Errorf("serviceConfig cannot be nil: %v", services)
		}
		// Check the Service
		if !IsValidService(serviceConfig.Service) {
			return fmt.Errorf("invalid service: %v", serviceConfig.Service)
		}
	}
	return nil
}

// ValidateSupplierServiceConfigs returns an error if any of the supplier service configs are invalid
func ValidateSupplierServiceConfigs(services []*sharedtypes.SupplierServiceConfig) error {
	if len(services) == 0 {
		return fmt.Errorf("no services provided for supplier: %v", services)
	}
	for _, serviceConfig := range services {
		if serviceConfig == nil {
			return fmt.Errorf("serviceConfig cannot be nil: %v", services)
		}

		// Check the Service
		if !IsValidService(serviceConfig.Service) {
			return fmt.Errorf("invalid service: %v", serviceConfig.Service)
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

			// Validate the URL
			if endpoint.Url == "" {
				return fmt.Errorf("endpoint.Url cannot be empty: %v", serviceConfig)
			}
			if !IsValidEndpointUrl(endpoint.Url) {
				return fmt.Errorf("invalid endpoint.Url: %v", serviceConfig)
			}

			// Validate the RPC type
			if endpoint.RpcType == sharedtypes.RPCType_UNKNOWN_RPC {
				return fmt.Errorf("endpoint.RpcType cannot be UNKNOWN_RPC: %v", serviceConfig)
			}
			if _, ok := sharedtypes.RPCType_name[int32(endpoint.RpcType)]; !ok {
				return fmt.Errorf("endpoint.RpcType is not a valid RPCType: %v", serviceConfig)
			}

			// TODO_MAINNET(@okdas)/TODO_DISCUSS: Either add validation for `endpoint.Configs` (can be a part of
			// `parseEndpointConfigs`), or change the config structure to be more clear about what is expected here
			// as currently, this is just a map[string]string, when values can be other types.
			// if endpoint.Configs == nil {
			// 	return fmt.Errorf("endpoint.Configs cannot be nil: %v", serviceConfig)
			// }
			// if len(endpoint.Configs) == 0 {
			// 	return fmt.Errorf("endpoint.Configs must have at least one entry: %v", serviceConfig)
			// }
		}
	}
	return nil
}
