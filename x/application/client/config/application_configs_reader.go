package config

import (
	"gopkg.in/yaml.v2"

	sharedhelpers "github.com/pokt-network/poktroll/x/shared/helpers"
)

// YAMLApplicationConfig is the structure describing a single service stake entry in the stake config file
type YAMLApplicationConfig struct {
	ServiceIds []string `yaml:"service_ids"`
}

// ParseSupplierServiceConfig parses the stake config file into a SupplierServiceConfig
func ParseApplicationConfigs(configContent []byte) ([]string, error) {
	var applicationServiceConfig YAMLApplicationConfig

	// Unmarshal the stake config file into a stakeConfig
	if err := yaml.Unmarshal(configContent, &applicationServiceConfig); err != nil {
		return nil, ErrApplicationConfigUnmarshalYAML.Wrapf("%s", err)
	}

	if len(applicationServiceConfig.ServiceIds) == 0 {
		return nil, ErrApplicationConfigEmptyContent
	}

	serviceIds := make([]string, 0, len(applicationServiceConfig.ServiceIds))
	for _, serviceId := range applicationServiceConfig.ServiceIds {
		// Validate serviceId
		if !sharedhelpers.IsValidServiceId(serviceId) {
			return nil, ErrApplicationConfigInvalidServiceId.Wrapf("%s", serviceId)
		}
		serviceIds = append(serviceIds, serviceId)
	}

	return serviceIds, nil
}
