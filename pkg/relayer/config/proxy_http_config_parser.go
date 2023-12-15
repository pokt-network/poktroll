package config

import "net/url"

// parseRelayMinerConfigs populates the proxy fields of the target structure that
// are relevant to the "http" type in the proxy section of the config file.
func parseHTTPProxyConfig(
	yamlProxyConfig YAMLRelayMinerProxyConfig,
	proxyConfig *RelayMinerProxyConfig,
) error {
	// Check if the proxy host is a valid URL
	proxyUrl, err := url.Parse(yamlProxyConfig.Host)
	if err != nil {
		return ErrRelayMinerConfigInvalidProxy.Wrapf(
			"invalid proxy host %s",
			err.Error(),
		)
	}

	proxyConfig.Host = proxyUrl.Host
	return nil
}

// parseRelayMinerConfigs populates the supplier fields of the target structure
// that are relevant to the "http" type in the supplier section of the config file.
func parseHTTPSupplierConfig(
	yamlSupplierServiceConfig YAMLRelayMinerSupplierServiceConfig,
	supplierServiceConfig *RelayMinerSupplierServiceConfig,
) error {
	var err error
	// Check if the supplier url is a valid URL
	supplierServiceConfig.Url, err = url.Parse(yamlSupplierServiceConfig.Url)
	if err != nil {
		return ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"invalid supplier url %s",
			err.Error(),
		)
	}

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
