package config

import (
	"net/url"

	"github.com/pokt-network/poktroll/pkg/polylog"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// HydrateSupplier populates a single supplier's fields of the RelayMinerConfig
// that are relevant to each supplier in the "suppliers" section of the config file.
func (supplierConfig *RelayMinerSupplierConfig) HydrateSupplier(
	logger polylog.Logger,
	yamlSupplierConfig YAMLRelayMinerSupplierConfig,
) error {
	logger = logger.With(
		"supplier_id", yamlSupplierConfig.ServiceId,
	)

	logger.Debug().Msgf("🔧 Starting to hydrate supplier configuration")

	// Supplier name is required
	if len(yamlSupplierConfig.ServiceId) == 0 {
		logger.Error().Msg("❌ Error hydrating supplier: supplier name is required")
		return ErrRelayMinerConfigInvalidSupplier.Wrap("supplier name is required")
	}
	supplierConfig.ServiceId = yamlSupplierConfig.ServiceId
	logger.Debug().Msgf("✅ Successfully set supplier service ID: %s", supplierConfig.ServiceId)

	// TODO_FUTURE: Consider validating signing key format/constraints here
	// Currently deferred to HydrateSuppliers() for inheritance from root config
	supplierConfig.SigningKeyNames = yamlSupplierConfig.SigningKeyNames
	if len(supplierConfig.SigningKeyNames) > 0 {
		logger.Debug().Msgf("✅ Successfully set signing key names: %v", supplierConfig.SigningKeyNames)
	} else {
		logger.Debug().Msg("🔑 No signing key names provided, will inherit from root config")
	}

	// Hydrate the default service config
	logger.Debug().Msg("🔧 Hydrating default service configuration")
	defaultServiceConfig, err := supplierConfig.hydrateServiceConfig(logger, yamlSupplierConfig.ServiceConfig)
	if err != nil {
		logger.Error().Msgf("❌ Error hydrating default service config: %v", err)
		return err
	}
	supplierConfig.ServiceConfig = defaultServiceConfig
	logger.Debug().Msg("✅ Successfully hydrated default service configuration")

	// Hydrate the RPC-type service-specific service configs (if any)
	supplierConfig.RPCTypeServiceConfigs = make(
		map[sharedtypes.RPCType]*RelayMinerSupplierServiceConfig,
		len(yamlSupplierConfig.RPCTypeServiceConfigs),
	)

	if len(yamlSupplierConfig.RPCTypeServiceConfigs) > 0 {
		logger.Debug().Msgf("🔧 Found %d RPC-type specific service configurations to hydrate", len(yamlSupplierConfig.RPCTypeServiceConfigs))
	}

	// Hydrate RPC-type specific configurations
	// Each RPC type (REST, JSON-RPC, etc.) can have its own backend and sett
	for rpcType, serviceConfig := range yamlSupplierConfig.RPCTypeServiceConfigs {
		logger.Debug().Msgf("🔧 Hydrating RPC-type specific config for: %s", rpcType)

		rpcType, err := sharedtypes.GetRPCTypeFromConfig(rpcType)
		if err != nil {
			logger.Error().Msgf("❌ Error getting RPC type from config: %v", err)
			return ErrRelayMinerConfigInvalidSupplier.Wrapf(
				"❌ Error getting RPC type from config: %q", err,
			)
		}

		rpcTypeServiceConfig, err := supplierConfig.hydrateServiceConfig(logger, serviceConfig)
		if err != nil {
			logger.Error().Msgf("❌ Error hydrating RPC-type specific service config for %s: %v", rpcType, err)
			return err
		}

		supplierConfig.RPCTypeServiceConfigs[rpcType] = rpcTypeServiceConfig
		logger.Debug().Msgf("✅ Successfully hydrated RPC-type specific config for: %s", rpcType)
	}

	logger.Debug().Msgf("🎉 Successfully completed hydration for supplier: %s", supplierConfig.ServiceId)
	return nil
}

// hydrateServiceConfig converts YAML service config to internal structure
// Validates backend URL, authentication, and headers configuration
func (supplierConfig *RelayMinerSupplierConfig) hydrateServiceConfig(
	logger polylog.Logger,
	supplierServiceConfigYAML YAMLRelayMinerSupplierServiceConfig,
) (*RelayMinerSupplierServiceConfig, error) {
	logger = logger.With(
		"backend_url", supplierServiceConfigYAML.BackendUrl,
	)

	logger.Debug().Msgf("🔧 Hydrating service config with backend URL: %s", supplierServiceConfigYAML.BackendUrl)

	backendUrl, err := url.Parse(supplierServiceConfigYAML.BackendUrl)
	if err != nil {
		logger.Error().Msgf("❌ Error parsing backend URL '%s': %v", supplierServiceConfigYAML.BackendUrl, err)
		return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"invalid supplier backend url %s",
			err.Error(),
		)
	}

	if backendUrl.Scheme == "" {
		logger.Error().Msgf("❌ Error: missing scheme in supplier backend URL: %s", supplierServiceConfigYAML.BackendUrl)
		return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"missing scheme in supplier backend url %s",
			supplierServiceConfigYAML.BackendUrl,
		)
	}

	logger.Debug().Msgf("✅ Successfully parsed backend URL with scheme: %s", backendUrl.Scheme)

	// Populate the supplier service fields that are relevant to each supported
	// supplier type.
	// If other supplier types are added in the future, they should be handled
	// by their own functions.
	supplierServiceConfig := &RelayMinerSupplierServiceConfig{}
	switch backendUrl.Scheme {
	case "http", "https", "ws", "wss":
		supplierConfig.ServerType = RelayMinerServerTypeHTTP
		logger.Debug().Msgf("🌐 Configuring HTTP/WebSocket server type for scheme: %s", backendUrl.Scheme)

		if err := supplierServiceConfig.
			parseSupplierBackendUrl(supplierServiceConfigYAML); err != nil {
			logger.Error().Msgf("❌ Error parsing supplier backend URL: %v", err)
			return nil, err
		}
		logger.Debug().Msg("✅ Successfully parsed supplier backend URL configuration")
	default:
		// Fail if the supplier type is not supported
		logger.Error().Msgf("❌ Error: unsupported supplier backend URL scheme: %s", backendUrl.Scheme)
		return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"invalid supplier type %s",
			backendUrl.Scheme,
		)
	}

	logger.Debug().Msg("🎉 Successfully completed service config hydration")
	return supplierServiceConfig, nil
}
