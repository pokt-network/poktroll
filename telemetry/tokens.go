package telemetry

import (
	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"
)

// TODO_MAINNET(@bryanchriswhite): Revisit how telemetry is managed under `x/tokenomics` to ensure that it
// complies with the new hardened settlement approach.

// TODO_MAINNET(@red-0ne, #897): Minted, burnt and slashd tokens values might not be completely accurate.
// While we're keeping this metric for now consider removing in favor of utilizing the `cosmos-exporter` which uses onchain data.
// Context: https://github.com/cosmos/cosmos-sdk/issues/21614, https://github.com/pokt-network/pocket/pull/832

// MintedTokensFromModule is a function to track token minting from a specific module.
// The metric used is an increment counter, and the label includes the module name for context.
func MintedTokensFromModule(module string, amount float32) {
	if !isTelemetyEnabled() {
		return
	}

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
