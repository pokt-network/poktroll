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

		// Check if the supplier exposed endpoint is a valid URL
		supplierHost, err := url.Parse(host)
		if err != nil {
			return ErrRelayMinerConfigInvalidSupplier.Wrapf(
				"invalid supplier public endpoint %s",
				host,
			)
		}

		// Check if the supplier public endpoint is unique
		if _, ok := existingEndpoints[supplierHost.Host]; ok {
			return ErrRelayMinerConfigInvalidSupplier.Wrapf(
				"duplicate supplier public endpoint %s",
				host,
			)
		}
		existingEndpoints[supplierHost.Host] = true

		// Add the supplier public endpoint to the suppliers list
		supplierConfig.PubliclyExposedEndpoints = append(
			supplierConfig.PubliclyExposedEndpoints,
			supplierHost.Host,
		)
	}

	// Add a default endpoint which corresponds to the supplier name if it is not
	// already in the list
	if _, ok := existingEndpoints[supplierConfig.ServiceId]; !ok {
		supplierConfig.PubliclyExposedEndpoints = append(
			supplierConfig.PubliclyExposedEndpoints,
			supplierConfig.ServiceId,
		)
	}

	// Populate the supplier service fields that are relevant to each supported
	// supplier type.
	// If other supplier types are added in the future, they should be handled
	// by their own functions.
	supplierConfig.ServiceConfig = &RelayMinerSupplierServiceConfig{}
	switch yamlSupplierConfig.ServerType {
	case "http":
		supplierConfig.ServerType = ServerTypeHTTP
		if err := supplierConfig.ServiceConfig.
			parseHTTPSupplierConfig(yamlSupplierConfig.ServiceConfig); err != nil {
			return err
		}
	default:
		// Fail if the supplier type is not supported
		return ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"invalid supplier type %s",
			yamlSupplierConfig.ServerType,
		)
	}

	return nil
}
