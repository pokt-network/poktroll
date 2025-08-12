package config

import (
	"time"

	"github.com/docker/go-units"
	yaml "gopkg.in/yaml.v2"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

// DefaultRequestTimeoutSeconds is the default timeout for requests in seconds.
// If not specified in the config, it will be used as a fallback.
// Using var to expose it for testing purposes.
const DefaultRequestTimeoutSeconds uint64 = 10

var DefaultRequestTimeoutDuration time.Duration = time.Duration(DefaultRequestTimeoutSeconds) * time.Second

// DefaultMaxBodySize defines the default maximum HTTP body size as a string, used as a fallback if unspecified.
const DefaultMaxBodySize = "20MB"

// ParseRelayMinerConfigs parses the relay miner config file into a RelayMinerConfig
func ParseRelayMinerConfigs(logger polylog.Logger, configContent []byte) (*RelayMinerConfig, error) {
	var (
		yamlRelayMinerConfig YAMLRelayMinerConfig
		relayMinerConfig     = &RelayMinerConfig{}
	)

	// The config file should not be empty
	if len(configContent) == 0 {
		return nil, ErrRelayMinerConfigEmpty
	}

	// Unmarshal the stake config file into a yamlRelayMinerConfig
	if err := yaml.Unmarshal(configContent, &yamlRelayMinerConfig); err != nil {
		return nil, ErrRelayMinerConfigUnmarshalYAML.Wrap(err.Error())
	}

	// Fallback to DefaultRequestTimeoutSeconds const if none is specified in the config
	// This is for the backwards compatibility with the previous versions of the config file
	if yamlRelayMinerConfig.DefaultRequestTimeoutSeconds == 0 {
		yamlRelayMinerConfig.DefaultRequestTimeoutSeconds = DefaultRequestTimeoutSeconds
	}

	if yamlRelayMinerConfig.DefaultMaxBodySize == "" {
		yamlRelayMinerConfig.DefaultMaxBodySize = DefaultMaxBodySize
	}

	size, err := units.RAMInBytes(yamlRelayMinerConfig.DefaultMaxBodySize)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidMaxBodySize.Wrapf(
			"invalid max body size %q",
			yamlRelayMinerConfig.DefaultMaxBodySize,
		)
	}
	relayMinerConfig.DefaultMaxBodySize = size

	// Global section
	relayMinerConfig.DefaultSigningKeyNames = yamlRelayMinerConfig.DefaultSigningKeyNames
	relayMinerConfig.DefaultRequestTimeoutSeconds = yamlRelayMinerConfig.DefaultRequestTimeoutSeconds

	// SmtStorePath is required
	if len(yamlRelayMinerConfig.SmtStorePath) == 0 {
		return nil, ErrRelayMinerConfigInvalidSmtStorePath
	}
	relayMinerConfig.SmtStorePath = yamlRelayMinerConfig.SmtStorePath

	// EnableOverServicing is a flag that indicates whether the relay miner
	// should enable over-servicing for the relays it serves.
	//
	// Over-servicing allows the offchain relay miner to serve more relays than the
	// amount of stake the onchain Application can pay the corresponding onchain
	// Supplier at the end of the session
	//
	// This can enable high quality of service for the network and earn quality points with Gateways.
	relayMinerConfig.EnableOverServicing = yamlRelayMinerConfig.EnableOverServicing

	// No additional validation on metrics. The server would fail to start if they are invalid
	// which is the intended behaviour.
	relayMinerConfig.Metrics = &RelayMinerMetricsConfig{
		Enabled: yamlRelayMinerConfig.Metrics.Enabled,
		Addr:    yamlRelayMinerConfig.Metrics.Addr,
	}

	relayMinerConfig.Pprof = &RelayMinerPprofConfig{
		Enabled: yamlRelayMinerConfig.Pprof.Enabled,
		Addr:    yamlRelayMinerConfig.Pprof.Addr,
	}

	relayMinerConfig.Ping = &RelayMinerPingConfig{
		Enabled: yamlRelayMinerConfig.Ping.Enabled,
		Addr:    yamlRelayMinerConfig.Ping.Addr,
	}

	// Hydrate the pocket node urls
	if err := relayMinerConfig.HydratePocketNodeUrls(&yamlRelayMinerConfig.PocketNode); err != nil {
		return nil, err
	}

	// Hydrate the relay miner servers config
	if err := relayMinerConfig.HydrateServers(yamlRelayMinerConfig.Suppliers); err != nil {
		return nil, err
	}

	// Hydrate the suppliers
	if err := relayMinerConfig.HydrateSuppliers(logger, yamlRelayMinerConfig.Suppliers); err != nil {
		return nil, err
	}

	return relayMinerConfig, nil
}
