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

		// Supplier name should be unique
		if _, ok := existingSuppliers[yamlSupplierConfig.ServiceId]; ok {
			return ErrRelayMinerConfigInvalidSupplier.Wrapf(
				"duplicate supplier name %s",
				yamlSupplierConfig.ServiceId,
			)
		}
		// Mark the supplier as existing
		existingSuppliers[yamlSupplierConfig.ServiceId] = true

		relayMinerConfig.
			Servers[yamlSupplierConfig.ListenUrl].
			Suppliers[supplierConfig.ServiceId] = supplierConfig
	}

	return nil
}
