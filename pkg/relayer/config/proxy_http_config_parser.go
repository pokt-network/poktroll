package config

import (
	"fmt"
	"net/url"
)

// parseHTTPProxyConfig populates the proxy fields of the target structure that
// are relevant to the "http" type in the proxy section of the config file.
// This function alters the target RelayMinerProxyConfig structure as a side effect.
func (proxyConfig *RelayMinerProxyConfig) parseHTTPProxyConfig(
	yamlProxyConfig YAMLRelayMinerProxyConfig,
) error {
	// Check if the proxy host is a valid URL.
	// Since `yamlProxyConfig.Host` is a string representing the host, we need to
	// prepend it with the "http://" scheme to make it a valid URL; we end up
	// using the `Host` field of the resulting `url.URL` struct, so the prepended
	// scheme is irrelevant.
	proxyUrl, err := url.Parse(fmt.Sprintf("http://%s", yamlProxyConfig.Host))
	if err != nil {
		return ErrRelayMinerConfigInvalidProxy.Wrapf(
			"invalid proxy host %s",
			err.Error(),
		)
	}

	if proxyUrl.Host == "" {
		return ErrRelayMinerConfigInvalidProxy.Wrap("empty proxy host")
	}

	proxyConfig.Host = proxyUrl.Host
	return nil
}

// parseHTTPSupplierConfig populates the supplier fields of the target structure
// that are relevant to the "http" type in the supplier section of the config file.
// This function alters the target RelayMinerSupplierServiceConfig structure
// as a side effect.
func (supplierServiceConfig *RelayMinerSupplierServiceConfig) parseHTTPSupplierConfig(
	yamlSupplierServiceConfig YAMLRelayMinerSupplierServiceConfig,
) error {
	// Check if the supplier url is not empty
	if len(yamlSupplierServiceConfig.Url) == 0 {
		return ErrRelayMinerConfigInvalidSupplier.Wrap("empty supplier url")
	}

	// Check if the supplier url is a valid URL
	supplierServiceUrl, err := url.Parse(yamlSupplierServiceConfig.Url)
	if err != nil {
		return ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"invalid supplier url %s",
			err.Error(),
		)
	}

	supplierServiceConfig.Url = supplierServiceUrl

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
