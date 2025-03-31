package telemetry

// Default configuration values for telemetry
const (
	// DefaultCardinalityLevel represents the default cardinality level for metrics collection
	DefaultCardinalityLevel = "medium"
)

// DefaultConfig returns the default telemetry configuration
func DefaultConfig() PocketTelemetryConfig {
	return PocketTelemetryConfig{
		CardinalityLevel: DefaultCardinalityLevel,
	}
}
