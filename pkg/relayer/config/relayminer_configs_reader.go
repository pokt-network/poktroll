package config

import yaml "gopkg.in/yaml.v2"

// ParseRelayMinerConfigs parses the relay miner config file into a RelayMinerConfig
func ParseRelayMinerConfigs(configContent []byte) (*RelayMinerConfig, error) {
	var (
		yamlRelayMinerConfig YAMLRelayMinerConfig
		relayMinerConfig     = &RelayMinerConfig{}
	)

	// The config file should not be empty
	if len(configContent) == 0 {
		return nil, ErrRelayMinerConfigEmpty
	}

	// Unmarshal the stake config file into a yamlAppGateConfig
	if err := yaml.Unmarshal(configContent, &yamlRelayMinerConfig); err != nil {
		return nil, ErrRelayMinerConfigUnmarshalYAML.Wrap(err.Error())
	}

	// Global section
	// SigningKeyName is required
	if len(yamlRelayMinerConfig.SigningKeyName) == 0 {
		return nil, ErrRelayMinerConfigInvalidSigningKeyName
	}
	relayMinerConfig.SigningKeyName = yamlRelayMinerConfig.SigningKeyName

	// SmtStorePath is required
	if len(yamlRelayMinerConfig.SmtStorePath) == 0 {
		return nil, ErrRelayMinerConfigInvalidSmtStorePath
	}
	relayMinerConfig.SmtStorePath = yamlRelayMinerConfig.SmtStorePath

	// No additional validation on metrics. The server would fail to start if they are invalid
	// which is the intended behaviour.
	relayMinerConfig.Metrics = &RelayMinerMetricsConfig{
		Enabled: yamlRelayMinerConfig.Metrics.Enabled,
		Addr:    yamlRelayMinerConfig.Metrics.Addr,
	}

	// Hydrate the pocket node urls
	if err := relayMinerConfig.HydratePocketNodeUrls(&yamlRelayMinerConfig.PocketNode); err != nil {
		return nil, err
	}

	// Hydrate the relay miner servers config
	if err := relayMinerConfig.HydrateServers(yamlRelayMinerConfig.Suppliers); err != nil {
		return nil, err
	}

	// Hydrate the suppliers
	if err := relayMinerConfig.HydrateSuppliers(yamlRelayMinerConfig.Suppliers); err != nil {
		return nil, err
	}

	return relayMinerConfig, nil
}
