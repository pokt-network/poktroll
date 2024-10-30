package telemetry

import (
	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"
)

// MetricNameKeys prefixes metrics with `poktroll` for easy identification.
// E.g., `("hodlers", "regret_level")` yields `poktroll_hodlers_regret_level` — great for tracking FOMO as hodlers rethink choices.
// Returns a slice of strings as `go-metric`, the underlying metrics library, expects.
func MetricNameKeys(metrics ...string) []string {
	result := make([]string, 0, len(metrics)+1)
	result = append(result, metricNamePrefix)
	result = append(result, metrics...)
	return result
}

// isTelemetyEnabled returns whether is telemetry turned on in the config file `app.toml` - cosmos-sdk's telemetry section.
func isTelemetyEnabled() bool {
	return cosmostelemetry.IsTelemetryEnabled()
}

// appendMediumCardinalityLabels only creates the label if cardinality if set to "medium" or higher.
// A good example for a "medium" cardinality use-case is `service_id`:
//   - This is a network wide parameter
//   - It is dependenon the permissionless nature of the network and can grow unbounded
//   - We're keeping an option to turn off such labels to avoid metric bloat
//
// Configuration option is exposed in app.toml under the `poktroll.telemetry` section.
func appendMediumCardinalityLabels(labels []metrics.Label, labelPairs ...metrics.Label) []metrics.Label {
	if globalTelemetryConfig.CardinalityLevel == "medium" || globalTelemetryConfig.CardinalityLevel == "high" {
		return append(labels, labelPairs...)
	}
	return labels
}

// appendHighCardinalityLabels only creates the label if cardinality is set to "high".
// A good example of high cardinality labels is `application_address` or `supplier_address`:
//   - This setting, on a large network, will slow down both the full node and the metric scraping system.
//   - These labels need to be exposed for local development, debugging and performance troubleshooting.
//
// Additional references on cardinality: https://www.robustperception.io/cardinality-is-key/
// Configuration option is exposed in app.toml under the `poktroll.telemetry` section.
func appendHighCardinalityLabels(labels []metrics.Label, labelPairs ...metrics.Label) []metrics.Label {
	if globalTelemetryConfig.CardinalityLevel == "high" {
		return append(labels, labelPairs...)
	}
	return labels
}

// toMetricLabel takes simple key and value of the label to return metrics.Label.
func toMetricLabel(key, value string) metrics.Label {
	return cosmostelemetry.NewLabel(key, value)
}
