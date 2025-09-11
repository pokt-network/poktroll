package config

import (
	"net/url"

	"github.com/docker/go-units"
)

// HydrateServers populates the servers fields of the RelayMinerConfig.
func (relayMinerConfig *RelayMinerConfig) HydrateServers(
	yamlSupplierConfigs []YAMLRelayMinerSupplierConfig,
) error {
	// At least one server is required
	if len(yamlSupplierConfigs) == 0 {
		return ErrRelayMinerConfigInvalidSupplier.Wrap("no suppliers provided")
	}

	relayMinerConfig.Servers = make(map[string]*RelayMinerServerConfig)

	for _, yamlSupplierConfig := range yamlSupplierConfigs {
		listenUrl, err := url.Parse(yamlSupplierConfig.ListenUrl)
		if err != nil {
			return ErrRelayMinerConfigInvalidServer.Wrapf(
				"invalid listen url %q",
				yamlSupplierConfig.ListenUrl,
			)
		}

		if listenUrl.Scheme == "" {
			return ErrRelayMinerConfigInvalidServer.Wrapf(
				"missing scheme in listen url %q",
				yamlSupplierConfig.ListenUrl,
			)
		}

		if _, ok := relayMinerConfig.Servers[yamlSupplierConfig.ListenUrl]; ok {
			continue
		}

		serverConfig := &RelayMinerServerConfig{
			XForwardedHostLookup:  yamlSupplierConfig.XForwardedHostLookup,
			SupplierConfigsMap:    make(map[string]*RelayMinerSupplierConfig),
			EnableEagerValidation: relayMinerConfig.EnableEagerValidation,
		}

		if yamlSupplierConfig.MaxBodySize == "" {
			serverConfig.MaxBodySize = relayMinerConfig.DefaultMaxBodySize
		} else {
			size, sizeErr := units.RAMInBytes(yamlSupplierConfig.MaxBodySize)
			if sizeErr != nil {
				return ErrRelayMinerConfigInvalidMaxBodySize.Wrapf(
					"invalid max body size %q",
					yamlSupplierConfig.MaxBodySize,
				)
			}
			serverConfig.MaxBodySize = size
		}

		// Populate the server fields that are relevant to each supported server type
		switch listenUrl.Scheme {
		case "http", "ws":
			if err := serverConfig.parseHTTPServerConfig(yamlSupplierConfig); err != nil {
				return err
			}
			serverConfig.ServerType = RelayMinerServerTypeHTTP
		default:
			// Fail if the relay miner server type is not supported
			return ErrRelayMinerConfigInvalidServer.Wrapf(
				"invalid relay miner server type %q",
				listenUrl.Scheme,
			)
		}

		relayMinerConfig.Servers[yamlSupplierConfig.ListenUrl] = serverConfig
	}

	return nil
}
