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

// isTelemetyEnabled returns whether is telemetry turned on in the config file.
func isTelemetyEnabled() bool {
	return cosmostelemetry.IsTelemetryEnabled()
}

// addMediumCardinalityLabel only creates the label if cardinality if set to "medium".
// Good example of a medium cardinality label is `service_id` — we do not control the number of services
// on the network, and as permissionless services grow the metrics can get easily out of hand. We're keeping
// an option to turn off such labels.
// Such labels are included when the cardinality is set to "high".
func addMediumCardinalityLabel(labels []metrics.Label, name string, value string) []metrics.Label {
	if globalTelemetryConfig.CardinalityLevel == "medium" || globalTelemetryConfig.CardinalityLevel == "high" {
		return append(labels, metrics.Label{Name: name, Value: value})
	}

	return labels
}

// addHighCardinalityLabel only creates the label if cardinality if set to "high".
// Good examples of high cardinality labels are `application_address` or `supplier_address`.
// This setting, on a large network, will slow down both the full node and the metric scraping system.
// We want to have such labels exposed for local development, debugging and performance troubleshooring.
// More background on why this is important: https://www.robustperception.io/cardinality-is-key/
func addHighCardinalityLabel(labels []metrics.Label, name string, value string) []metrics.Label {
	if globalTelemetryConfig.CardinalityLevel == "high" {
		return append(labels, metrics.Label{Name: name, Value: value})
	}

	return labels
}
