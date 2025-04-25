package config

import "net/url"

// parseHTTPServerConfig populates the server fields of the target structure that
// are relevant to the "http" type.
// This function alters the target RelayMinerServerConfig structure as a side effect.
func (serverConfig *RelayMinerServerConfig) parseHTTPServerConfig(
	yamlSupplierConfig YAMLRelayMinerSupplierConfig,
) error {
	// Validate yamlSupplierConfig.ListenUrl.
	listenUrl, err := url.Parse(yamlSupplierConfig.ListenUrl)
	if err != nil {
		return ErrRelayMinerConfigInvalidServer.Wrapf(
			"invalid relay miner server listen address %s",
			err.Error(),
		)
	}

	// Ensure that the host is not empty and use it as the server listen address.
	if listenUrl.Host == "" {
		return ErrRelayMinerConfigInvalidServer.Wrap("empty relay miner server listen address")
	}

	serverConfig.ListenAddress = listenUrl.Host
	return nil
}

// parseSupplierBackendUrl populates the supplier fields of the target structure
// that are relevant to "http" and "https" backend url service configurations.
// This function alters the target RelayMinerSupplierServiceConfig structure
// as a side effect.
func (supplierServiceConfig *RelayMinerSupplierServiceConfig) parseSupplierBackendUrl(
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
	supplierServiceConfig.ForwardPocketHeaders = yamlSupplierServiceConfig.ForwardIdentityHeaders

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
