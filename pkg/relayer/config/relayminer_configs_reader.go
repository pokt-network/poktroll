package config

import (
	"os"
	"path/filepath"
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

// DefaultMinedRelaysStorePath is the default path for the mined relays storage.
// It is used when the deprecated :memory: or :memory_pebble: values are found in the config.
const DefaultMinedRelaysStorePath = ".pocket/smt"

// Deprecated SMT store path values that should be replaced with the default path
const (
	DeprecatedSmtStorePathMemory       = ":memory:"
	DeprecatedSmtStorePathMemoryPebble = ":memory_pebble:"
)

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
		return nil, ErrRelayMinerConfigInvalidSmtStorePath.Wrapf("smt store path is: '%s'", yamlRelayMinerConfig.SmtStorePath)
	}

	relayMinerConfig.SmtStorePath = yamlRelayMinerConfig.SmtStorePath

	// Handle deprecated :memory: and :memory_pebble: entries for backwards compatibility
	if relayMinerConfig.SmtStorePath == DeprecatedSmtStorePathMemory ||
		relayMinerConfig.SmtStorePath == DeprecatedSmtStorePathMemoryPebble {

		// Get the home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, ErrRelayMinerConfigInvalidSmtStorePath.Wrapf(
				"failed to get home directory for default SMT store path: %v", err,
			)
		}

		// Use the default path: $HOME/.pocket/smt
		defaultPath := filepath.Join(homeDir, DefaultMinedRelaysStorePath)
		relayMinerConfig.SmtStorePath = defaultPath

		// Log deprecation warning
		logger.Warn().
			Str("deprecated_value", yamlRelayMinerConfig.SmtStorePath).
			Str("fallback_path", defaultPath).
			Msg("Deprecated smt_store_path value detected. Using default persistent storage path. Please update your config file.")
	}

	// EnableOverServicing is a flag that indicates whether the relay miner
	// should enable over-servicing for the relays it serves.
	//
	// Over-servicing allows the offchain relay miner to serve more relays than the
	// amount of stake the onchain Application can pay the corresponding onchain
	// Supplier at the end of the session
	//
	// This can enable high quality of service for the network and earn quality points with Gateways.
	relayMinerConfig.EnableOverServicing = yamlRelayMinerConfig.EnableOverServicing

	// EnableEagerValidation is a flag that indicates whether the relay miner
	// should enable eager validation for all incoming relay requests.
	//
	// When enabled, all incoming relay requests are validated immediately upon receipt.
	// When disabled, relay requests are validated only if their session is known,
	// or validation is deferred if their session is unknown.
	relayMinerConfig.EnableEagerRelayRequestValidation = yamlRelayMinerConfig.EnableEagerRelayRequestValidation

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
