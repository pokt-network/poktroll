package config

import (
	"fmt"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// DefaultRequestTimeoutSeconds is the default timeout for requests in seconds.
// If not specified in the config, it will be used as a fallback.
const DefaultRequestTimeoutSeconds = 10

const DefaultMaxBodySize = int64(20 * 1024 * 1024)
const DefaultMaxBodySizeStr = "20MB"

func parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	unitMultipliers := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}

	// Extract numeric part and unit
	numPart := ""
	unitPart := ""
	for i, c := range sizeStr {
		if c < '0' || c > '9' {
			numPart = sizeStr[:i]
			unitPart = strings.ToUpper(strings.TrimSpace(sizeStr[i:]))
			break
		}
	}

	// Convert the numeric part to int64
	numValue, err := strconv.Atoi(numPart)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	// Apply the unit multiplier
	multiplier, ok := unitMultipliers[unitPart]
	if !ok {
		return 0, fmt.Errorf("unsupported size unit: %s", unitPart)
	}

	// Calculate and return the final size in bytes
	return int64(numValue) * multiplier, nil
}

// ParseRelayMinerConfigs parses the relay miner config file into a RelayMinerConfig
func ParseRelayMinerConfigs(configContent []byte) (*RelayMinerConfig, error) {
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
		yamlRelayMinerConfig.DefaultMaxBodySize = DefaultMaxBodySizeStr
	}

	size, err := parseSize(yamlRelayMinerConfig.DefaultMaxBodySize)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidMaxBodySize
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
	if err := relayMinerConfig.HydrateSuppliers(yamlRelayMinerConfig.Suppliers); err != nil {
		return nil, err
	}

	return relayMinerConfig, nil
}
