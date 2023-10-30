package helpers

import (
	"fmt"

	sharedtypes "pocket/x/shared/types"
)

// AreValidAppServiceConfigs returns an error if the provided service configs are invalid
// by wrapping the provided around with additional details
func AreValidAppServiceConfigs(services []*sharedtypes.ApplicationServiceConfig) (string, bool) {
	if len(services) == 0 {
		return fmt.Sprintf("no services configs provided for application: %v", services), false
	}
	for _, serviceConfig := range services {
		if serviceConfig == nil {
			return fmt.Sprintf("serviceConfig cannot be nil: %v", services), false
		}
		if serviceConfig.ServiceId == nil {
			return fmt.Sprintf("serviceId cannot be nil: %v", serviceConfig), false
		}
		if serviceConfig.ServiceId.Id == "" {
			return fmt.Sprintf("serviceId.Id cannot be empty: %v", serviceConfig), false
		}
		if !IsValidServiceId(serviceConfig.ServiceId.Id) {
			return fmt.Sprintf("invalid serviceId.Id: %v", serviceConfig), false
		}
		if !IsValidServiceName(serviceConfig.ServiceId.Name) {
			return fmt.Sprintf("invalid serviceId.Name: %v", serviceConfig), false
		}
	}
	return "", true
}
