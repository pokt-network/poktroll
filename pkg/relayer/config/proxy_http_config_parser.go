package config

import "net/url"

// parseHTTPProxyConfig populates the proxy fields of the target structure that
// are relevant to the "http" type in the proxy section of the config file.
// This function alters the target RelayMinerProxyConfig structure as a side effect.
func (proxyConfig *RelayMinerProxyConfig) parseHTTPProxyConfig(
	yamlSupplierConfig YAMLRelayMinerSupplierConfig,
) error {
	// Check if the proxy listen address is a valid URL.
	// Since `yamlProxyConfig.ListenAddress` is a string representing the host,
	// we need to prepend it with the "http://" scheme to make it a valid URL;
	// we end up using the `Host` field of the resulting `url.URL` struct,
	// so the prepended scheme is irrelevant.
	listenUrl, err := url.Parse(yamlSupplierConfig.ListenUrl)
	if err != nil {
		return ErrRelayMinerConfigInvalidProxy.Wrapf(
			"invalid proxy listen address %s",
			err.Error(),
		)
	}

	if listenUrl.Host == "" {
		return ErrRelayMinerConfigInvalidProxy.Wrap("empty proxy listen address")
	}

	proxyConfig.ListenAddress = listenUrl.Host
	return nil
}

// parseHTTPSupplierConfig populates the supplier fields of the target structure
// that are relevant to the "http" type in the supplier section of the config file.
// This function alters the target RelayMinerSupplierServiceConfig structure
// as a side effect.
func (supplierServiceConfig *RelayMinerSupplierServiceConfig) parseHTTPSupplierConfig(
	yamlSupplierServiceConfig YAMLRelayMinerSupplierServiceConfig,
) error {
	// Check if the supplier backend url is empty
	if len(yamlSupplierServiceConfig.BackendUrl) == 0 {
		return ErrRelayMinerConfigInvalidSupplier.Wrap("empty supplier backend url")
	}

	// Check if the supplier backend url is a valid URL
	supplierServiceBackendUrl, err := url.Parse(yamlSupplierServiceConfig.BackendUrl)
	if err != nil {
		return ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"invalid supplier backend url %s",
			err.Error(),
		)
	}

	supplierServiceConfig.BackendUrl = supplierServiceBackendUrl

	// If the Authentication section is not empty, populate the supplier service
	// authentication fields
	if yamlSupplierServiceConfig.Authentication != (YAMLRelayMinerSupplierServiceAuthentication{}) {
		supplierServiceConfig.Authentication = &RelayMinerSupplierServiceAuthentication{
			Username: yamlSupplierServiceConfig.Authentication.Username,
			Password: yamlSupplierServiceConfig.Authentication.Password,
		}
	}

	// If the Headers section is not empty, populate the supplier service headers fields
	if yamlSupplierServiceConfig.Headers != nil {
		supplierServiceConfig.Headers = yamlSupplierServiceConfig.Headers
	}

	return nil
}
