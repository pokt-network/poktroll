package telemetry

// MetricNameKeys constructs the full metric name by prefixing with a defined
// prefix and appending any additional metrics provided as variadic arguments.
func MetricNameKeys(metrics ...string) []string {
	result := make([]string, 0, len(metrics)+1)
	result = append(result, metricNamePrefix)
	result = append(result, metrics...)
	return result
}
