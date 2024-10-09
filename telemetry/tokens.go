package telemetry

import (
	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"
)

// MintedTokensFromModule is a function to track token minting from a specific module.
// The metric used is an increment counter, and the label includes the module name for context.
func MintedTokensFromModule(module string, amount float32) {
	if !isTelemetyEnabled() {
		return
	}

	// CosmosSDK has a metric called `minted_tokens` (as a part of `mint` module), however it is wrongfully marked a `gauge`.
	// It should be an `increment` because it always goes up. `gauge` tracks data that can go up and down.
	// More info: https://prometheus.io/docs/concepts/metric_types/
	//
	// We can't keep the same metric name because different metric types can't collide under the same name. So we add
	// `poktroll_` prefix instead.
	cosmostelemetry.IncrCounterWithLabels(
		MetricNameKeys("minted", "tokens"),
		amount,
		[]metrics.Label{
			cosmostelemetry.NewLabel("module", module),
		},
	)
}

// BurnedTokensFromModule is a function to track token burning from a specific module.
// The metric used is an increment counter, and the label includes the module name for context.
func BurnedTokensFromModule(module string, amount float32) {
	if !isTelemetyEnabled() {
		return
	}

	cosmostelemetry.IncrCounterWithLabels(
		MetricNameKeys("burned", "tokens"),
		amount,
		[]metrics.Label{
			cosmostelemetry.NewLabel("module", module),
		},
	)
}

// SlashedTokensFromModule is a function to track token slashing from a specific module.
// The metric used is an increment counter, and the label includes the module name for context.
func SlashedTokensFromModule(module string, amount float32) {
	if !isTelemetyEnabled() {
		return
	}

	cosmostelemetry.IncrCounterWithLabels(
		MetricNameKeys("slashed", "tokens"),
		amount,
		[]metrics.Label{
			cosmostelemetry.NewLabel("module", module),
		},
	)
}
