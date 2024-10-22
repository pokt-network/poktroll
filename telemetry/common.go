package telemetry

import (
	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"
)

// MetricNameKeys constructs the full metric name by prefixing with a defined
// prefix and appending any additional metrics provided as variadic arguments.
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

// appendMediumCardinalityLabels only creates the label if cardinality if set to "medium".
// Good example of a medium cardinality label is `service_id` — we do not control the number of services
// on the network, and as permissionless services grow the metrics can get easily out of hand. We're keeping
// an option to turn off such labels.
// Medium cardinality labels are included when the cardinality is set to "high".
// Configuration option is exposed in app.toml, our own `poktroll.telemetry` section.
func appendMediumCardinalityLabels(labels []metrics.Label, labelPairs ...metrics.Label) []metrics.Label {
	if globalTelemetryConfig.CardinalityLevel == "medium" || globalTelemetryConfig.CardinalityLevel == "high" {
		return append(labels, labelPairs...)
	}
	return labels
}

// appendHighCardinalityLabels only creates the label if cardinality if set to "high".
// Good examples of high cardinality labels are `application_address` or `supplier_address`.
// This setting, on a large network, will slow down both the full node and the metric scraping system.
// We want to have such labels exposed for local development, debugging and performance troubleshooring.
// More background on why this is important: https://www.robustperception.io/cardinality-is-key/
// Configuration option is exposed in app.toml, our own `poktroll.telemetry` section.
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
