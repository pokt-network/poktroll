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

	// Supplier hosts
	supplierConfig.Hosts = []string{}
	existingHosts := make(map[string]bool)
	for _, host := range yamlSupplierConfig.Hosts {
		// Check if the supplier host is empty
		if len(host) == 0 {
			return ErrRelayMinerConfigInvalidSupplier.Wrap("empty supplier host")
		}

		// Check if the supplier host is a valid URL
		supplierHost, err := url.Parse(host)
		if err != nil {
			return ErrRelayMinerConfigInvalidSupplier.Wrapf(
				"invalid supplier host %s",
				host,
			)
		}

		// Check if the supplier host is unique
		if _, ok := existingHosts[supplierHost.Host]; ok {
			return ErrRelayMinerConfigInvalidSupplier.Wrapf(
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
	if _, ok := existingHosts[supplierConfig.ServiceId]; !ok {
		supplierConfig.Hosts = append(supplierConfig.Hosts, supplierConfig.ServiceId)
	}

	// Populate the supplier service fields that are relevant to each supported
	// supplier type.
	// If other supplier types are added in the future, they should be handled
	// by their own functions.
	supplierConfig.ServiceConfig = &RelayMinerSupplierServiceConfig{}
	switch yamlSupplierConfig.Type {
	case "http":
		supplierConfig.Type = ProxyTypeHTTP
		if err := supplierConfig.ServiceConfig.
			parseHTTPSupplierConfig(yamlSupplierConfig.ServiceConfig); err != nil {
			return err
		}
	default:
		// Fail if the supplier type is not supported
		return ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"invalid supplier type %s",
			yamlSupplierConfig.Type,
		)
	}

	// Check if the supplier has proxies
	if len(yamlSupplierConfig.ProxyNames) == 0 {
		return ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"supplier %s has no proxies",
			supplierConfig.ServiceId,
		)
	}

	return nil
}
