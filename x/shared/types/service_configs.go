package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	requiredRevSharePercentageSum = uint64(100)
)

// ValidateAppServiceConfigs returns an error if any of the application service configs are invalid
func ValidateAppServiceConfigs(services []*ApplicationServiceConfig) error {
	if len(services) != 1 {
		return fmt.Errorf("application must have exactly one service: %v", services)
	}
	for _, serviceConfig := range services {
		if serviceConfig == nil {
			return fmt.Errorf("serviceConfig cannot be nil: %v", services)
		}
		// Check the Service ID
		if !IsValidServiceId(serviceConfig.GetServiceId()) {
			return ErrSharedInvalidService.Wrapf("invalid service ID: %q", serviceConfig.GetServiceId())
		}
	}
	return nil
}

// ValidateSupplierServiceConfigs returns an error if any of the supplier service configs are invalid
func ValidateSupplierServiceConfigs(services []*SupplierServiceConfig) error {
	if len(services) == 0 {
		return fmt.Errorf("no services provided for supplier: %v", services)
	}
	for _, serviceConfig := range services {
		if serviceConfig == nil {
			return fmt.Errorf("serviceConfig cannot be nil: %v", services)
		}

		// Check the Service ID
		if !IsValidServiceId(serviceConfig.GetServiceId()) {
			return ErrSharedInvalidService.Wrapf("invalid service ID: %s", serviceConfig.GetServiceId())
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
			if endpoint.RpcType == RPCType_UNKNOWN_RPC {
				return fmt.Errorf("endpoint.RpcType cannot be UNKNOWN_RPC: %v", serviceConfig)
			}
			if _, ok := RPCType_name[int32(endpoint.RpcType)]; !ok {
				return fmt.Errorf("endpoint.RpcType is not a valid RPCType: %v", serviceConfig)
			}

			// TODO_MAINNET(@okdas): Either add validation for `endpoint.Configs` (can be a part of
			// `parseEndpointConfigs`), or change the config structure to be more clear about what is expected here
			// as currently, this is just a map[string]string, when values can be other types.
			// if endpoint.Configs == nil {
			// 	return fmt.Errorf("endpoint.Configs cannot be nil: %v", serviceConfig)
			// }
			// if len(endpoint.Configs) == 0 {
			// 	return fmt.Errorf("endpoint.Configs must have at least one entry: %v", serviceConfig)
			// }
		}

		if err := ValidateServiceRevShare(serviceConfig.RevShare); err != nil {
			return err
		}
	}

	return nil
}

// ValidateServiceRevShare validates the supplier's service revenue share,
// ensuring that the sum of the revenue share percentages is 100.
// NB: This function is unit tested via the supplier staking config tests.
func ValidateServiceRevShare(revShareList []*ServiceRevenueShare) error {
	revSharePercentageSum := uint64(0)

	if len(revShareList) == 0 {
		return ErrSharedInvalidRevShare.Wrap("no rev share configurations")
	}

	for _, revShare := range revShareList {
		if revShare == nil {
			return ErrSharedInvalidRevShare.Wrap("rev share cannot be nil")
		}

		// Validate the revenue share address
		if revShare.Address == "" {
			return ErrSharedInvalidRevShare.Wrapf("rev share address cannot be empty: %v", revShare)
		}

		if _, err := sdk.AccAddressFromBech32(revShare.Address); err != nil {
			return ErrSharedInvalidRevShare.Wrapf("invalid rev share address %s; (%v)", revShare.Address, err)
		}

		if revShare.RevSharePercentage <= 0 || revShare.RevSharePercentage > 100 {
			return ErrSharedInvalidRevShare.Wrapf(
				"invalid rev share value %v; must be between 0 and 100",
				revShare.RevSharePercentage,
			)
		}

		revSharePercentageSum += revShare.RevSharePercentage
	}

	if revSharePercentageSum != requiredRevSharePercentageSum {
		return ErrSharedInvalidRevShare.Wrapf(
			"invalid rev share percentage sum %v; must be equal to %v",
			revSharePercentageSum,
			requiredRevSharePercentageSum,
		)
	}

	return nil
}
