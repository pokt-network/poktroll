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

		// If SigningKeyNames are not specified for this supplierConfig, we want
		// the supplier to use the default list from the root of the config.
		if len(supplierConfig.SigningKeyNames) == 0 || supplierConfig.SigningKeyNames == nil {
			// If neither lists are specified - we need to throw an error to let
			// user configure the signing keys.
			if len(relayMinerConfig.DefaultSigningKeyNames) == 0 || relayMinerConfig.DefaultSigningKeyNames == nil {
				return ErrRelayMinerConfigInvalidSigningKeyName.Wrapf(
					"'default_signing_key_names' is not configured and 'signing_key_names' is empty for the supplier %s",
					yamlSupplierConfig.ServiceId,
				)
			}

			// Otherwise assign the DefaultSigningKeyNames to this supplier.
			supplierConfig.SigningKeyNames = relayMinerConfig.DefaultSigningKeyNames
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
			SupplierConfigsMap[supplierConfig.ServiceId] = supplierConfig
	}

	return nil
}
