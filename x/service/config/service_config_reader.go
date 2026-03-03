package config

import (
	"gopkg.in/yaml.v2"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// YAMLEditServiceConfig is the top-level structure describing the edit-service
// config file containing one or more service entries.
type YAMLEditServiceConfig struct {
	Services []*YAMLServiceEntry `yaml:"services"`
}

// YAMLServiceEntry is the structure describing a single service entry in the
// edit-service config file.
type YAMLServiceEntry struct {
	ServiceId            string `yaml:"service_id"`
	ServiceName          string `yaml:"service_name"`
	ComputeUnitsPerRelay uint64 `yaml:"compute_units_per_relay"`
}

// ParseEditServiceConfig parses the YAML config content into a YAMLEditServiceConfig.
// It validates that each entry has a non-empty service_id and compute_units_per_relay > 0.
// The service_name field is optional; when omitted the CLI will use the on-chain name
// since the chain does not support updating service names for existing services.
// Ref: x/service/keeper/msg_server_add_service.go:55-65
func ParseEditServiceConfig(configContent []byte) (*YAMLEditServiceConfig, error) {
	if len(configContent) == 0 {
		return nil, ErrServiceConfigEmptyContent
	}

	var editConfig YAMLEditServiceConfig
	if err := yaml.Unmarshal(configContent, &editConfig); err != nil {
		return nil, ErrServiceConfigUnmarshalYAML.Wrapf("%s", err)
	}

	if len(editConfig.Services) == 0 {
		return nil, ErrServiceConfigNoServices
	}

	for _, svc := range editConfig.Services {
		if err := sharedtypes.IsValidServiceId(svc.ServiceId); err != nil {
			return nil, ErrServiceConfigInvalidServiceId.Wrapf("%s", err)
		}

		// service_name is optional for edit-service; the on-chain name is used when omitted.

		if svc.ComputeUnitsPerRelay == 0 {
			return nil, ErrServiceConfigInvalidComputeUnits.Wrapf(
				"service_id: %s; compute_units_per_relay must be greater than 0",
				svc.ServiceId,
			)
		}
	}

	return &editConfig, nil
}
