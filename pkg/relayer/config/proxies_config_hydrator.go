package config

import "net/url"

// HydrateProxies populates the proxies fields of the RelayMinerConfig that
// are relevant to the "proxies" section in the config file.
func (relayMinerConfig *RelayMinerConfig) HydrateProxies(
	yamlSupplierConfigs []YAMLRelayMinerSupplierConfig,
) error {
	// At least one proxy is required
	if len(yamlSupplierConfigs) == 0 {
		return ErrRelayMinerConfigInvalidSupplier.Wrap("no suppliers provided")
	}

	relayMinerConfig.Proxies = make(map[string]*RelayMinerProxyConfig)

	for _, yamlSupplierConfig := range yamlSupplierConfigs {
		listenUrl, err := url.Parse(yamlSupplierConfig.ListenUrl)
		if err != nil {
			return ErrRelayMinerConfigInvalidProxy.Wrapf(
				"invalid listen url %s",
				yamlSupplierConfig.ListenUrl,
			)
		}

		if listenUrl.Scheme == "" {
			return ErrRelayMinerConfigInvalidProxy.Wrapf(
				"missing scheme in listen url %s",
				yamlSupplierConfig.ListenUrl,
			)
		}

		proxyName := yamlSupplierConfig.ListenUrl

		if _, ok := relayMinerConfig.Proxies[proxyName]; ok {
			continue
		}

		proxyConfig := &RelayMinerProxyConfig{
			ProxyName:            proxyName,
			XForwardedHostLookup: yamlSupplierConfig.XForwardedHostLookup,
			Suppliers:            make(map[string]*RelayMinerSupplierConfig),
		}

		// Populate the proxy fields that are relevant to each supported proxy type
		switch listenUrl.Scheme {
		case "http":
			if err := proxyConfig.parseHTTPProxyConfig(yamlSupplierConfig); err != nil {
				return err
			}
			proxyConfig.ServerType = ServerTypeHTTP
		default:
			// Fail if the proxy type is not supported
			return ErrRelayMinerConfigInvalidProxy.Wrapf(
				"invalid proxy type %s",
				listenUrl.Scheme,
			)
		}

		relayMinerConfig.Proxies[proxyConfig.ProxyName] = proxyConfig
	}

	return nil
}
