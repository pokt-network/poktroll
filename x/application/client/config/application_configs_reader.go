package config

import (
	"gopkg.in/yaml.v2"

	sharedhelpers "github.com/pokt-network/poktroll/x/shared/helpers"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// YAMLApplicationConfig is the structure describing a single service stake entry in the stake config file
// TODO_DOCUMENT(@red-0ne): Add additional documentation on app config files.
type YAMLApplicationConfig struct {
	ServiceIds []string `yaml:"service_ids"`
}

// ParseApplicationConfig parses the stake config file and returns a slice of ApplicationServiceConfig
func ParseApplicationConfigs(configContent []byte) ([]*sharedtypes.ApplicationServiceConfig, error) {
	var parsedAppConfig YAMLApplicationConfig

	// Unmarshal the stake config file into a applicationServiceConfig
	if err := yaml.Unmarshal(configContent, &parsedAppConfig); err != nil {
		return nil, ErrApplicationConfigUnmarshalYAML.Wrapf("%s", err)
	}

	if len(parsedAppConfig.ServiceIds) == 0 {
		return nil, ErrApplicationConfigEmptyContent
	}

	// Prepare the applicationServiceConfig
	applicationServiceConfig := make(
		[]*sharedtypes.ApplicationServiceConfig,
		0,
		len(parsedAppConfig.ServiceIds),
	)

	for _, serviceId := range parsedAppConfig.ServiceIds {
		// Validate serviceId
		if !sharedhelpers.IsValidServiceId(serviceId) {
			return nil, ErrApplicationConfigInvalidServiceId.Wrapf("%s", serviceId)
		}

		appServiceConfig := &sharedtypes.ApplicationServiceConfig{
			Service: &sharedtypes.Service{Id: serviceId},
		}

		applicationServiceConfig = append(applicationServiceConfig, appServiceConfig)
	}

	return applicationServiceConfig, nil
}
