package config

import (
	"net/url"

	yaml "gopkg.in/yaml.v2"
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
	if len(yamlRelayMinerConfig.SigningKeyName) == 0 {
		return nil, ErrRelayMinerConfigInvalidSigningKeyName
	}

	// SmtStorePath is required
	if len(yamlRelayMinerConfig.SmtStorePath) == 0 {
		return nil, ErrRelayMinerConfigInvalidSmtStorePath
	}

	// Pocket node urls section
	relayMinerPocketConfig := &RelayMinerPocketNodeConfig{}

	if len(yamlRelayMinerConfig.PocketNode.TxNodeGRPCUrl) == 0 {
		return nil, ErrRelayMinerConfigInvalidNodeUrl.Wrap("tx node grpc url is required")
	}

	// Check if the pocket node grpc url is a valid URL
	txNodeGRPCUrl, err := url.Parse(yamlRelayMinerConfig.PocketNode.TxNodeGRPCUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidNodeUrl.Wrapf(
			"invalid tx node grpc url %s",
			err.Error(),
		)
	}
	relayMinerPocketConfig.TxNodeGRPCUrl = txNodeGRPCUrl

	// If the query node grpc url is empty, use the tx node grpc url
	if len(yamlRelayMinerConfig.PocketNode.QueryNodeGRPCUrl) == 0 {
		relayMinerPocketConfig.QueryNodeGRPCUrl = relayMinerPocketConfig.TxNodeGRPCUrl
	} else {
		// If the query node grpc url is not empty, make sure it is a valid URL
		queryNodeGRPCUrl, err := url.Parse(yamlRelayMinerConfig.PocketNode.QueryNodeGRPCUrl)
		if err != nil {
			return nil, ErrRelayMinerConfigInvalidNodeUrl.Wrapf(
				"invalid query node grpc url %s",
				err.Error(),
			)
		}
		relayMinerPocketConfig.QueryNodeGRPCUrl = queryNodeGRPCUrl
	}

	if len(yamlRelayMinerConfig.PocketNode.QueryNodeRPCUrl) == 0 {
		return nil, ErrRelayMinerConfigInvalidNodeUrl.Wrap("query node rpc url is required")
	}

	// Check if the query node rpc url is a valid URL
	queryNodeRPCUrl, err := url.Parse(yamlRelayMinerConfig.PocketNode.QueryNodeRPCUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidNodeUrl.Wrapf(
			"invalid query node rpc url %s",
			err.Error(),
		)
	}
	relayMinerPocketConfig.QueryNodeRPCUrl = queryNodeRPCUrl

	// Proxies section
	// At least one proxy is required
	if len(yamlRelayMinerConfig.Proxies) == 0 {
		return nil, ErrRelayMinerConfigInvalidProxy.Wrap("no proxies provided")
	}

	relayMinerProxiesConfig := make(map[string]*RelayMinerProxyConfig)

	for _, yamlProxyConfig := range yamlRelayMinerConfig.Proxies {
		// Proxy name is required
		if len(yamlProxyConfig.Name) == 0 {
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
			Name:                 yamlProxyConfig.Name,
			XForwardedHostLookup: yamlProxyConfig.XForwardedHostLookup,
			Suppliers:            make(map[string]*RelayMinerSupplierConfig),
		}

		// Populate the proxy fields that are relevant to each supported proxy type
		switch yamlProxyConfig.Type {
		case "http":
			if err := proxyConfig.parseHTTPProxyConfig(yamlProxyConfig); err != nil {
				return nil, err
			}
		default:
			// Fail if the proxy type is not supported
			return nil, ErrRelayMinerConfigInvalidProxy.Wrapf(
				"invalid proxy type %s",
				yamlProxyConfig.Type,
			)
		}

		switch yamlProxyConfig.Type {
		case "http":
			proxyConfig.Type = ProxyTypeHTTP
		default:
			ErrRelayMinerConfigInvalidProxy.Wrapf(
				"invalid proxy type %s",
				yamlProxyConfig.Type,
			)
		}

		relayMinerProxiesConfig[proxyConfig.Name] = proxyConfig
	}

	// Suppliers section
	relayMinerSuppliersConfig := make(map[string]*RelayMinerSupplierConfig)

	for _, yamlSupplierConfig := range yamlRelayMinerConfig.Suppliers {
		// Supplier name is required
		if len(yamlSupplierConfig.Name) == 0 {
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
			// Check if the supplier host is empty
			if len(host) == 0 {
				return nil, ErrRelayMinerConfigInvalidSupplier.Wrap("empty supplier host")
			}

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
			if err := supplierConfig.ServiceConfig.
				parseHTTPSupplierConfig(yamlSupplierConfig.ServiceConfig); err != nil {
				return nil, err
			}
		default:
			// Fail if the supplier type is not supported
			return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
				"invalid supplier type %s",
				yamlSupplierConfig.Type,
			)
		}

		switch yamlSupplierConfig.Type {
		case "http":
			supplierConfig.Type = ProxyTypeHTTP
		default:
			ErrRelayMinerConfigInvalidProxy.Wrapf(
				"invalid proxy type %s",
				yamlSupplierConfig.Type,
			)
		}

		// Check if the supplier has proxies
		if len(yamlSupplierConfig.ProxyNames) == 0 {
			return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
				"supplier %s has no proxies",
				supplierConfig.Name,
			)
		}

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

	// Check if a proxy is referencing a host more than once
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
		PocketNode:     relayMinerPocketConfig,
		SigningKeyName: yamlRelayMinerConfig.SigningKeyName,
		SmtStorePath:   yamlRelayMinerConfig.SmtStorePath,
		Proxies:        relayMinerProxiesConfig,
	}

	return relayMinerCMDConfig, nil
}
