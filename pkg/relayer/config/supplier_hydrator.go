package config

import (
	"net/url"
	"strings"

	"github.com/pokt-network/poktroll/x/shared/types"
)

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

	// Hydrate the default service config
	defaultServiceConfig, err := supplierConfig.hydrateServiceConfig(yamlSupplierConfig.ServiceConfig)
	if err != nil {
		return err
	}
	supplierConfig.ServiceConfig = defaultServiceConfig

	// Hydrate the RPC type-specific service configs (if any)
	supplierConfig.RPCTypeServiceConfigs = make(
		map[types.RPCType]*RelayMinerSupplierServiceConfig,
		len(yamlSupplierConfig.RPCTypeServiceConfigs),
	)

	// Loop through the RPC type-specific service configs and hydrate them.
	// For example, if the supplier is configured to handle REST and JSON-RPC,
	// there will be two RPC type-specific service configs.
	for rpcType, serviceConfig := range yamlSupplierConfig.RPCTypeServiceConfigs {
		rpcType, err := getRPCTypeFromConfig(rpcType)
		if err != nil {
			return err
		}

		rpcTypeServiceConfig, err := supplierConfig.hydrateServiceConfig(serviceConfig)
		if err != nil {
			return err
		}

		supplierConfig.RPCTypeServiceConfigs[rpcType] = rpcTypeServiceConfig
	}

	return nil
}

// getRPCTypeFromConfig converts the string RPC type to the
// types.RPCType enum and performs validation.
//
// eg. "rest" -> types.RPCType_REST
func getRPCTypeFromConfig(rpcType string) (types.RPCType, error) {
	rpcTypeInt, ok := types.RPCType_value[strings.ToUpper(rpcType)]
	if !ok {
		return 0, ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"invalid rpc type %s",
			rpcType,
		)
	}
	if !rpcTypeIsValid(types.RPCType(rpcTypeInt)) {
		return 0, ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"invalid rpc type %s",
			rpcType,
		)
	}
	return types.RPCType(rpcTypeInt), nil
}

// rpcTypeIsValid checks if the RPC type is valid.
// It is used to validate the RPC type-specific service configs.
func rpcTypeIsValid(rpcType types.RPCType) bool {
	switch rpcType {
	case types.RPCType_GRPC,
		types.RPCType_WEBSOCKET,
		types.RPCType_JSON_RPC,
		types.RPCType_REST,
		types.RPCType_HYBRID:
		return true
	default:
		return false
	}
}

// hydrateServiceConfig hydrates a single service config by parsing the
// YAMLRelayMinerSupplierServiceConfig and populating the RelayMinerSupplierServiceConfig
// structure. It returns the populated RelayMinerSupplierServiceConfig and an error
// if the service config is invalid.
func (supplierConfig *RelayMinerSupplierConfig) hydrateServiceConfig(
	supplierServiceConfigYAML YAMLRelayMinerSupplierServiceConfig,
) (*RelayMinerSupplierServiceConfig, error) {
	backendUrl, err := url.Parse(supplierServiceConfigYAML.BackendUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"invalid supplier backend url %s",
			err.Error(),
		)
	}

	if backendUrl.Scheme == "" {
		return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"missing scheme in supplier backend url %s",
			supplierServiceConfigYAML.BackendUrl,
		)
	}

	// Populate the supplier service fields that are relevant to each supported
	// supplier type.
	// If other supplier types are added in the future, they should be handled
	// by their own functions.
	supplierServiceConfig := &RelayMinerSupplierServiceConfig{}
	switch backendUrl.Scheme {
	case "http", "https", "ws", "wss":
		supplierConfig.ServerType = RelayMinerServerTypeHTTP
		if err := supplierServiceConfig.
			parseSupplierBackendUrl(supplierServiceConfigYAML); err != nil {
			return nil, err
		}
	default:
		// Fail if the supplier type is not supported
		return nil, ErrRelayMinerConfigInvalidSupplier.Wrapf(
			"invalid supplier type %s",
			backendUrl.Scheme,
		)
	}

	return supplierServiceConfig, nil
}
