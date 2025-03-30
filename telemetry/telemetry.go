package telemetry

import (
	"fmt"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/mitchellh/mapstructure"
)

// globalTelemetryConfig stores pocket specific telemetry configurations.
// Set once on initialization and remains constant during runtime.
var globalTelemetryConfig PocketTelemetryConfig

// PocketTelemetryConfig represents the telemetry portion of the custom pocket config section in `app.toml`.
type PocketTelemetryConfig struct {
	CardinalityLevel string `mapstructure:"cardinality-level"`
}

// New sets the globalTelemetryConfig for telemetry package.
func New(appOpts servertypes.AppOptions) error {
	// Get the pocket config section. If it doesn't exist, use defaults
	pocketConfig := appOpts.Get("pocket")
	if pocketConfig == nil {
		globalTelemetryConfig = DefaultConfig()
		return nil
	}

	// Try to get the telemetry subsection
	pocketMap, ok := pocketConfig.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid pocket config format: expected map[string]interface{}, got %T", pocketConfig)
	}

	telemetryMap, ok := pocketMap["telemetry"].(map[string]interface{})
	if !ok {
		globalTelemetryConfig = DefaultConfig()
		return nil
	}

	// Use mapstructure to decode the map into the struct
	if err := mapstructure.Decode(telemetryMap, &globalTelemetryConfig); err != nil {
		return fmt.Errorf("error decoding pocket.telemetry config: %v", err)
	}

	return nil
}
