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

	// Top level section
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

	// Hydrate the pocket node urls
	if err := relayMinerConfig.HydratePocketNodeUrls(&yamlRelayMinerConfig.PocketNode); err != nil {
		return nil, err
	}

	// Hydrate the proxies
	if err := relayMinerConfig.HydrateProxies(yamlRelayMinerConfig.Proxies); err != nil {
		return nil, err
	}

	// Hydrate the suppliers
	if err := relayMinerConfig.HydrateSuppliers(yamlRelayMinerConfig.Suppliers); err != nil {
		return nil, err
	}

	// Check if proxies are referencing hosts more than once
	if err := relayMinerConfig.EnsureUniqueHosts(); err != nil {
		return nil, err
	}

	return relayMinerConfig, nil
}

// EnsureUniqueHosts checks if each proxy is referencing a host more than once
func (relayMinerConfig *RelayMinerConfig) EnsureUniqueHosts() error {
	for _, proxyConfig := range relayMinerConfig.Proxies {
		existingHosts := make(map[string]bool)
		for _, supplierConfig := range proxyConfig.Suppliers {
			for _, host := range supplierConfig.Hosts {
				if _, ok := existingHosts[host]; ok {
					return ErrRelayMinerConfigInvalidProxy.Wrapf(
						"duplicate host %s in proxy %s",
						host,
						proxyConfig.Name,
					)
				}
				existingHosts[host] = true
			}
		}
	}

	return nil
}
