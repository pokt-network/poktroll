package telemetry

import (
	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"
)

// TODO: Compare the metrics here with the `cosmos-exporter` which uses onchain data.
// Refs:
// - https://github.com/cosmos/cosmos-sdk/issues/21614
// - https://github.com/pokt-network/poktroll/pull/832

// MintedTokensFromModule counts how many tokens are minted to a module account.
func MintedTokensFromModule(module string, amount float32) {
	if !isTelemetryEnabled() {
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

// BurnedTokensFromModule counts how many tokens are burnt from a module account.
func BurnedTokensFromModule(module string, amount float32) {
	if !isTelemetryEnabled() {
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

// SlashedTokensFromModule counts how many tokens are slashed from a module account.
func SlashedTokensFromModule(module string, amount float32) {
	if !isTelemetryEnabled() {
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
