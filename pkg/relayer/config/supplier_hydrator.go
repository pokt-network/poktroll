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

	// NB: Intentionally not verifying SigningKeyNames here.
	// We'll copy the keys from the root config in `HydrateSuppliers` if this list is empty.
	// `HydrateSuppliers` is a part of `pkg/relayer/config/suppliers_config_hydrator.go`.
	supplierConfig.SigningKeyNames = yamlSupplierConfig.SigningKeyNames

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
