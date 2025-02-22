package config

import "net/url"

// HydrateSupplier populates a single supplier's fields of the RelayMinerConfig
// that are relevant to each supplier in the "suppliers" section of the config file.
func (supplierConfig *RelayMinerSupplierConfig) HydrateSupplier(
	yamlSupplierConfig YAMLRelayMinerSupplierConfig,
) error {
	// Supplier name is required
	if len(yamlSupplierConfig.ServiceId) == 0 {
		return ErrRelayMinerConfigInvalidSupplier.Wrap("supplier name is required")
	}
	supplierConfig.ServiceId = yamlSupplierConfig.ServiceId

	// Supplier public endpoints
	supplierConfig.PubliclyExposedEndpoints = []string{}
	existingEndpoints := make(map[string]bool)
	for _, host := range yamlSupplierConfig.ServiceConfig.PubliclyExposedEndpoints {
		// Check if the supplier host is empty
		if len(host) == 0 {
			return ErrRelayMinerConfigInvalidSupplier.Wrap("empty supplier public endpoint")
		}

		// Check if the supplier public endpoint is unique
		if _, ok := existingEndpoints[host]; ok {
			return ErrRelayMinerConfigInvalidSupplier.Wrapf(
				"duplicate supplier public endpoint %s",
				host,
			)
		}
		existingEndpoints[host] = true

		// Add the supplier public endpoint to the suppliers list
		supplierConfig.PubliclyExposedEndpoints = append(
			supplierConfig.PubliclyExposedEndpoints,
			host,
		)
	}

	// NB: Intentionally not verifying SigningKeyNames here.
	// We'll copy the keys from the root config in `HydrateSuppliers` if this list is empty.
	// `HydrateSuppliers` is a part of `pkg/relayer/config/suppliers_config_hydrator.go`.
	supplierConfig.SigningKeyNames = yamlSupplierConfig.SigningKeyNames

	// Add a default endpoint which corresponds to the supplier name if it is not
	// already in the list
	if _, ok := existingEndpoints[supplierConfig.ServiceId]; !ok {
		supplierConfig.PubliclyExposedEndpoints = append(
			supplierConfig.PubliclyExposedEndpoints,
			supplierConfig.ServiceId,
		)
	}

	backendUrl, err := url.Parse(yamlSupplierConfig.ServiceConfig.BackendUrl)
	if err != nil {
		return ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"invalid supplier backend url %s",
			err.Error(),
		)
	}

	if backendUrl.Scheme == "" {
		return ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"missing scheme in supplier backend url %s",
			yamlSupplierConfig.ServiceConfig.BackendUrl,
		)
	}

	// Populate the supplier service fields that are relevant to each supported
	// supplier type.
	// If other supplier types are added in the future, they should be handled
	// by their own functions.
	supplierConfig.ServiceConfig = &RelayMinerSupplierServiceConfig{}
	switch backendUrl.Scheme {
	case "http", "https", "ws", "wss":
		supplierConfig.ServerType = RelayMinerServerTypeHTTP
		if err := supplierConfig.ServiceConfig.
			parseSupplierBackendUrl(yamlSupplierConfig.ServiceConfig); err != nil {
			return err
		}
	default:
		// Fail if the supplier type is not supported
		return ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"invalid supplier type %s",
			backendUrl.Scheme,
		)
	}

	return nil
}
