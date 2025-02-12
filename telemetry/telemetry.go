package telemetry

import (
	"fmt"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/mitchellh/mapstructure"
)

// globalTelemetryConfig stores poktroll specific telemetry configurations.
// Set once on initialization and remains constant during runtime.
var globalTelemetryConfig PoktrollTelemetryConfig

// PoktrollTelemetryConfig represents the telemetry portion of the custom poktroll config section in `app.toml`.
type PoktrollTelemetryConfig struct {
	CardinalityLevel string `mapstructure:"cardinality-level"`
}

// New sets the globalTelemetryConfig for telemetry package.
func New(appOpts servertypes.AppOptions) error {
	// Get the poktroll config section. If it doesn't exist, use defaults
	poktrollConfig := appOpts.Get("poktroll")
	if poktrollConfig == nil {
		globalTelemetryConfig = DefaultConfig()
		return nil
	}

	// Try to get the telemetry subsection
	poktrollMap, ok := poktrollConfig.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid poktroll config format: expected map[string]interface{}, got %T", poktrollConfig)
	}

	telemetryMap, ok := poktrollMap["telemetry"].(map[string]interface{})
	if !ok {
		globalTelemetryConfig = DefaultConfig()
		return nil
	}

	// Use mapstructure to decode the map into the struct
	if err := mapstructure.Decode(telemetryMap, &globalTelemetryConfig); err != nil {
		return fmt.Errorf("error decoding poktroll.telemetry config: %v", err)
	}

	return nil
}
