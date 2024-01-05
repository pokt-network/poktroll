package config

// HydrateSuppliers populates the suppliers fields of the RelayMinerConfig that
// are relevant to the "suppliers" section in the config file.
func (relayMinerConfig *RelayMinerConfig) HydrateSuppliers(
	yamlSupplierConfigs []YAMLRelayMinerSupplierConfig,
) error {
	existingSuppliers := make(map[string]bool)
	for _, yamlSupplierConfig := range yamlSupplierConfigs {
		// Hydrate and validate each supplier in the suppliers list of the config file.
		supplierConfig := &RelayMinerSupplierConfig{}
		if err := supplierConfig.HydrateSupplier(yamlSupplierConfig); err != nil {
			return err
		}

		// Supplier name should not be unique
		if _, ok := existingSuppliers[yamlSupplierConfig.ServiceId]; ok {
			return ErrRelayMinerConfigInvalidSupplier.Wrapf(
				"duplicate supplier name %s",
				yamlSupplierConfig.ServiceId,
			)
		}
		// Mark the supplier as existing
		existingSuppliers[yamlSupplierConfig.ServiceId] = true

		// Add the supplier config to the referenced proxies
		for _, proxyName := range yamlSupplierConfig.ProxyNames {
			// If the proxy name is referencing a non-existent proxy, fail
			if _, ok := relayMinerConfig.Proxies[proxyName]; !ok {
				return ErrRelayMinerConfigInvalidSupplier.Wrapf(
					"no matching proxy %s for supplier %s",
					supplierConfig.ServiceId,
					proxyName,
				)
			}

			// If the proxy name is referencing a proxy of a different type, fail
			if supplierConfig.Type != relayMinerConfig.Proxies[proxyName].Type {
				return ErrRelayMinerConfigInvalidSupplier.Wrapf(
					"supplier %s and proxy %s have different types",
					supplierConfig.ServiceId,
					proxyName,
				)
			}

			relayMinerConfig.Proxies[proxyName].Suppliers[supplierConfig.ServiceId] = supplierConfig
		}
	}

	return nil
}
