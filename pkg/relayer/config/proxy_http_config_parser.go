package config

import "net/url"

// parseHTTPProxyConfig populates the proxy fields of the target structure that
// are relevant to the "http" type.
// This function alters the target RelayMinerProxyConfig structure as a side effect.
func (proxyConfig *RelayMinerProxyConfig) parseHTTPProxyConfig(
	yamlSupplierConfig YAMLRelayMinerSupplierConfig,
) error {
	// Validate yamlSupplierConfig.ListenUrl.
	listenUrl, err := url.Parse(yamlSupplierConfig.ListenUrl)
	if err != nil {
		return ErrRelayMinerConfigInvalidProxy.Wrapf(
			"invalid proxy listen address %s",
			err.Error(),
		)
	}

	// Ensure that the host is not empty and use it as the server listen address.
	if listenUrl.Host == "" {
		return ErrRelayMinerConfigInvalidProxy.Wrap("empty proxy listen url")
	}

	proxyConfig.ListenAddress = listenUrl.Host
	return nil
}

// parseHTTPSupplierConfig populates the supplier fields of the target structure
// that are relevant to the "http".
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
