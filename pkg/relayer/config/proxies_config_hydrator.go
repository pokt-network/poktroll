package config

// HydrateProxies populates the proxies fields of the RelayMinerConfig that
// are relevant to the "proxies" section in the config file.
func (relayMinerConfig *RelayMinerConfig) HydrateProxies(
	yamlProxyConfigs []YAMLRelayMinerProxyConfig,
) error {
	// At least one proxy is required
	if len(yamlProxyConfigs) == 0 {
		return ErrRelayMinerConfigInvalidProxy.Wrap("no proxies provided")
	}

	relayMinerConfig.Proxies = make(map[string]*RelayMinerProxyConfig)

	for _, yamlProxyConfig := range yamlProxyConfigs {
		// Proxy name is required
		if len(yamlProxyConfig.ProxyName) == 0 {
			return ErrRelayMinerConfigInvalidProxy.Wrap("proxy name is required")
		}

		// Proxy name should not be unique
		if _, ok := relayMinerConfig.Proxies[yamlProxyConfig.ProxyName]; ok {
			return ErrRelayMinerConfigInvalidProxy.Wrapf(
				"duplicate porxy name %s",
				yamlProxyConfig.ProxyName,
			)
		}

		proxyConfig := &RelayMinerProxyConfig{
			ProxyName:            yamlProxyConfig.ProxyName,
			XForwardedHostLookup: yamlProxyConfig.XForwardedHostLookup,
			Suppliers:            make(map[string]*RelayMinerSupplierConfig),
		}

		// Populate the proxy fields that are relevant to each supported proxy type
		switch yamlProxyConfig.Type {
		case "http":
			if err := proxyConfig.parseHTTPProxyConfig(yamlProxyConfig); err != nil {
				return err
			}
		default:
			// Fail if the proxy type is not supported
			return ErrRelayMinerConfigInvalidProxy.Wrapf(
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

		relayMinerConfig.Proxies[proxyConfig.ProxyName] = proxyConfig
	}

	return nil
}
