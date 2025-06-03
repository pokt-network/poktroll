package config

import (
	yaml "gopkg.in/yaml.v2"
)

// DefaultRequestTimeoutSeconds is the default timeout for requests in seconds.
// If not specified in the config, it will be used as a fallback.
const DefaultRequestTimeoutSeconds = 30

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

	// Unmarshal the stake config file into a yamlRelayMinerConfig
	if err := yaml.Unmarshal(configContent, &yamlRelayMinerConfig); err != nil {
		return nil, ErrRelayMinerConfigUnmarshalYAML.Wrap(err.Error())
	}

	if yamlRelayMinerConfig.DefaultRequestTimeoutSeconds < 0 {
		return nil, ErrRelayMinerConfigInvalidRequestTimeout.Wrapf(
			"invalid 'default_request_timeout_seconds' value %d, it must be greater than or equal to 0",
			yamlRelayMinerConfig.DefaultRequestTimeoutSeconds,
		)
	}

	// Fallback to DefaultRequestTimeoutSeconds const if none is specified in the config
	// This is to keeps the backwards compatibility with the previous versions of the config file
	if yamlRelayMinerConfig.DefaultRequestTimeoutSeconds == 0 {
		yamlRelayMinerConfig.DefaultRequestTimeoutSeconds = DefaultRequestTimeoutSeconds
	}

	// Global section
	relayMinerConfig.DefaultSigningKeyNames = yamlRelayMinerConfig.DefaultSigningKeyNames
	relayMinerConfig.DefaultRequestTimeoutSeconds = yamlRelayMinerConfig.DefaultRequestTimeoutSeconds

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

	relayMinerConfig.Pprof = &RelayMinerPprofConfig{
		Enabled: yamlRelayMinerConfig.Pprof.Enabled,
		Addr:    yamlRelayMinerConfig.Pprof.Addr,
	}

	relayMinerConfig.Ping = &RelayMinerPingConfig{
		Enabled: yamlRelayMinerConfig.Ping.Enabled,
		Addr:    yamlRelayMinerConfig.Ping.Addr,
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
