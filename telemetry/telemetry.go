package telemetry

import (
	"fmt"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/mitchellh/mapstructure"
)

// globalTelemetryConfig stores poktroll specific telemetry configurations.
// It is set on initialization and does not change for the lifetime of the program.
var globalTelemetryConfig PoktrollTelemetryConfig

// PoktrollTelemetryConfig represents the telemetry protion of the custom poktroll config section in `app.toml`.
type PoktrollTelemetryConfig struct {
	CardinalityLevel string `mapstructure:"cardinality-level"`
}

// New sets the globalTelemetryConfig for telemetry package.
func New(appOpts servertypes.AppOptions) error {
	// Extract the map from appOpts.
	// `poktroll.telemetry` comes from `app.toml` which is parsed into a map.
	telemetryMap := appOpts.Get("poktroll.telemetry").(map[string]interface{})

	// Use mapstructure to decode the map into the struct
	if err := mapstructure.Decode(telemetryMap, &globalTelemetryConfig); err != nil {
		return fmt.Errorf("error decoding poktroll.telemetry config: %v", err)
	}

	return nil
}
