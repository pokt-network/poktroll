package config

import (
	"net/url"

	"gopkg.in/yaml.v2"
)

// ParseRelayMinerConfigs parses the relay miner config file into a RelayMinerConfig
func ParseRelayMinerConfigs(configContent []byte) (*RelayMinerConfig, error) {
	var (
		yamlRelayMinerConfig YAMLRelayMinerConfig
		err                  error
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
	if yamlRelayMinerConfig.SigningKeyName == "" {
		return nil, ErrRelayMinerConfigInvalidSigningKeyName
	}

	// SmtStorePath is required
	if yamlRelayMinerConfig.SmtStorePath == "" {
		return nil, ErrRelayMinerConfigInvalidSmtStorePath
	}

	// Pocket node urls section
	relayMinerPocketConfig := &RelayMinerPocketConfig{}
	pocket := yamlRelayMinerConfig.Pocket

	// Check if the pocket node grpc url is a valid URL
	relayMinerPocketConfig.TxNodeGRPCUrl, err = url.Parse(pocket.TxNodeGRPCUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidNodeUrl.Wrapf(
			"invalid tx node grpc url %s",
			err.Error(),
		)
	}

	// If the query node grpc url is empty, use the tx node grpc url
	if pocket.QueryNodeGRPCUrl == "" {
		relayMinerPocketConfig.QueryNodeGRPCUrl = relayMinerPocketConfig.TxNodeGRPCUrl
	} else {
		// If the query node grpc url is not empty, make sure it is a valid URL
		relayMinerPocketConfig.QueryNodeGRPCUrl, err = url.Parse(pocket.QueryNodeGRPCUrl)
		if err != nil {
			return nil, ErrRelayMinerConfigInvalidNodeUrl.Wrapf(
				"invalid query node grpc url %s",
				err.Error(),
			)
		}
	}

	// Check if the query node rpc url is a valid URL
	relayMinerPocketConfig.QueryNodeRPCUrl, err = url.Parse(pocket.QueryNodeRPCUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidNodeUrl.Wrapf(
			"invalid query node rpc url %s",
			err.Error(),
		)
	}

	// Proxies section
	// At least one proxy is required
	if len(yamlRelayMinerConfig.Proxies) == 0 {
		return nil, ErrRelayMinerConfigInvalidProxy.Wrap("no proxies provided")
	}

	proxies := yamlRelayMinerConfig.Proxies
	relayMinerProxiesConfig := make(map[string]*RelayMinerProxyConfig)

	for _, yamlProxyConfig := range proxies {
		// Proxy name is required
		if yamlProxyConfig.Name == "" {
			return nil, ErrRelayMinerConfigInvalidProxy.Wrap("proxy name is required")
		}

		// Proxy name should not be unique
		if _, ok := relayMinerProxiesConfig[yamlProxyConfig.Name]; ok {
			return nil, ErrRelayMinerConfigInvalidProxy.Wrapf(
				"duplicate porxy name %s",
				yamlProxyConfig.Name,
			)
		}

		proxyConfig := &RelayMinerProxyConfig{
			Name:      yamlProxyConfig.Name,
			Suppliers: make(map[string]*RelayMinerSupplierConfig),
		}

		// Populate the proxy fields that are relevant to each supported proxy type
		switch yamlProxyConfig.Type {
		case "http":
			if err := parseHTTPProxyConfig(yamlProxyConfig, proxyConfig); err != nil {
				return nil, err
			}
		default:
			// Fail if the proxy type is not supported
			return nil, ErrRelayMinerConfigInvalidProxy.Wrapf(
				"invalid proxy type %s",
				yamlProxyConfig.Type,
			)
		}
		proxyConfig.Type = yamlProxyConfig.Type

		relayMinerProxiesConfig[proxyConfig.Name] = proxyConfig
	}

	// Suppliers section
	suppliers := yamlRelayMinerConfig.Suppliers
	relayMinerSuppliersConfig := make(map[string]*RelayMinerSupplierConfig)

	for _, yamlSupplierConfig := range suppliers {
		// Supplier name is required
		if yamlSupplierConfig.Name == "" {
			return nil, ErrRelayMinerConfigInvalidSupplier.Wrap("supplier name is required")
		}

		// Supplier name should not be unique
		if _, ok := relayMinerSuppliersConfig[yamlSupplierConfig.Name]; ok {
			return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
				"duplicate supplier name %s",
				yamlSupplierConfig.Name,
			)
		}

		supplierConfig := &RelayMinerSupplierConfig{
			Name:          yamlSupplierConfig.Name,
			Hosts:         []string{},
			ServiceConfig: &RelayMinerSupplierServiceConfig{},
		}

		// Supplier hosts sub-section
		existingHosts := make(map[string]bool)
		for _, host := range yamlSupplierConfig.Hosts {
			// Check if the supplier host is a valid URL
			supplierHost, err := url.Parse(host)
			if err != nil {
				return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
					"invalid supplier host %s",
					host,
				)
			}

			// Check if the supplier host is unique
			if _, ok := existingHosts[supplierHost.Host]; ok {
				return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
					"duplicate supplier host %s",
					host,
				)
			}
			existingHosts[supplierHost.Host] = true

			// Add the supplier host to the suppliers list
			supplierConfig.Hosts = append(supplierConfig.Hosts, supplierHost.Host)
		}

		// Add a default host which corresponds to the supplier name if it is not
		// already in the list
		if _, ok := existingHosts[supplierConfig.Name]; !ok {
			supplierConfig.Hosts = append(supplierConfig.Hosts, supplierConfig.Name)
		}

		// Supplier service sub-section
		// Populate the supplier service fields that are relevant to each supported
		// supplier type.
		// If other supplier types are added in the future, they should be handled
		// by their own functions.
		switch yamlSupplierConfig.Type {
		case "http":
			if err := parseHTTPSupplierConfig(
				yamlSupplierConfig.ServiceConfig,
				supplierConfig.ServiceConfig,
			); err != nil {
				return nil, err
			}
		default:
			// Fail if the supplier type is not supported
			return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
				"invalid supplier type %s",
				yamlSupplierConfig.Type,
			)
		}
		supplierConfig.Type = yamlSupplierConfig.Type

		// Add the supplier config to the referenced proxies
		for _, proxyName := range yamlSupplierConfig.ProxyNames {
			// If the proxy name is referencing a non-existent proxy, fail
			if _, ok := relayMinerProxiesConfig[proxyName]; !ok {
				return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
					"no matching proxy %s for supplier %s",
					supplierConfig.Name,
					proxyName,
				)
			}

			// If the proxy name is referencing a proxy of a different type, fail
			if supplierConfig.Type != relayMinerProxiesConfig[proxyName].Type {
				return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
					"supplier %s and proxy %s have different types",
					supplierConfig.Name,
					proxyName,
				)
			}

			relayMinerProxiesConfig[proxyName].Suppliers[supplierConfig.Name] = supplierConfig
		}
	}

	// Check that a proxy is not referencing a host more than once
	for _, proxyConfig := range relayMinerProxiesConfig {
		existingHosts := make(map[string]bool)
		for _, supplierConfig := range proxyConfig.Suppliers {
			for _, host := range supplierConfig.Hosts {
				if _, ok := existingHosts[host]; ok {
					return nil, ErrRelayMinerConfigInvalidProxy.Wrapf(
						"duplicate host %s in proxy %s",
						host,
						proxyConfig.Name,
					)
				}
				existingHosts[host] = true
			}
		}
	}

	// Populate the relay miner config
	relayMinerCMDConfig := &RelayMinerConfig{
		SigningKeyName: yamlRelayMinerConfig.SigningKeyName,
		SmtStorePath:   yamlRelayMinerConfig.SmtStorePath,
		Pocket:         relayMinerPocketConfig,
		Proxies:        relayMinerProxiesConfig,
	}

	return relayMinerCMDConfig, nil
}
