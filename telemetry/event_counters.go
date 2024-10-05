// Package telemetry provides a set of functions for incrementing counters which track
// various events across the codebase. Typically, calls to these counter functions SHOULD
// be made inside deferred anonymous functions so that they will reference the final values
// of their inputs. Any instrumented piece of code which contains branching logic with respect
// its counter function inputs is subject to this constraint (i.e. MUST defer).
package telemetry

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

const (
	// Prefix all metric names with "poktroll" for easier search
	metricNamePrefix = "poktroll"

	// Label Names
	applicationAddressLabelName      = "app_addr"
	supplierOperatorAddressLabelName = "supop_addr"
)

// EventSuccessCounter increments a counter with the given data type and success status.
func EventSuccessCounter(
	eventType string,
	getValue func() float32,
	isSuccessful func() bool,
) {
	if !isTelemetyEnabled() {
		return
	}

	value := getValue()

	var metricName []string

	if isSuccessful() {
		metricName = MetricNameKeys("successful", "events")
	} else {
		metricName = MetricNameKeys("failed", "events")
	}

	telemetry.IncrCounterWithLabels(
		metricName,
		value,
		[]metrics.Label{
			{Name: "type", Value: eventType},
		},
	)
}

// ProofRequirementCounter increments a counter which tracks the number of claims
// which require proof for the given proof requirement reason (i.e. not required,
// probabilistic selection, above compute unit threshold).
// If err is not nil, the counter is not incremented but Prometheus will ingest this event.
func ProofRequirementCounter(
	reason prooftypes.ProofRequirementReason,
	serviceId string,
	applicationAddress string,
	supplierOperatorAddress string,
	err error,
) {
	if !isTelemetyEnabled() {
		return
	}

	incrementAmount := 1
	labels := []metrics.Label{
		{Name: "reason", Value: reason.String()},
	}
	labels = addMediumCardinalityLabel(labels, "service_id", serviceId)
	labels = addHighCardinalityLabel(labels, applicationAddressLabelName, applicationAddress)
	labels = addHighCardinalityLabel(labels, supplierOperatorAddressLabelName, supplierOperatorAddress)

	// Ensure the counter is not incremented if there was an error.
	if err != nil {
		return
	}

	telemetry.IncrCounterWithLabels(
		MetricNameKeys("proof", "requirements"),
		float32(incrementAmount),
		labels,
	)
}

// ClaimComputeUnitsCounter increments a counter which tracks the number of compute units
// which are represented by on-chain claims at the given ClaimProofStage.
// If err is not nil, the counter is not incremented but Prometheus will ingest this event.
func ClaimComputeUnitsCounter(
	claimProofStage prooftypes.ClaimProofStage,
	numComputeUnits uint64,
	serviceId string,
	applicationAddress string,
	supplierOperatorAddress string,
	err error,
) {
	if !isTelemetyEnabled() {
		return
	}

	incrementAmount := numComputeUnits
	labels := []metrics.Label{
		{Name: "proof_stage", Value: claimProofStage.String()},
	}
	labels = addMediumCardinalityLabel(labels, "service_id", serviceId)
	labels = addHighCardinalityLabel(labels, applicationAddressLabelName, applicationAddress)
	labels = addHighCardinalityLabel(labels, supplierOperatorAddressLabelName, supplierOperatorAddress)

	// Ensure the counter is not incremented if there was an error.
	if err != nil {
		incrementAmount = 0
	}

	telemetry.IncrCounterWithLabels(
		MetricNameKeys("compute_units"),
		float32(incrementAmount),
		labels,
	)
}

// ClaimRelaysCounter increments a counter which tracks the number of relays
// represented by on-chain claims at the given ClaimProofStage.
// If err is not nil, the counter is not incremented and an "error" label is added
// with the error's message. I.e., Prometheus will ingest this event.
func ClaimRelaysCounter(
	claimProofStage prooftypes.ClaimProofStage,
	numRelays uint64,
	serviceId string,
	applicationAddress string,
	supplierOperatorAddress string,
	err error,
) {
	if !isTelemetyEnabled() {
		return
	}

	incrementAmount := numRelays
	labels := []metrics.Label{
		{Name: "proof_stage", Value: claimProofStage.String()},
	}
	labels = addMediumCardinalityLabel(labels, "service_id", serviceId)
	labels = addHighCardinalityLabel(labels, applicationAddressLabelName, applicationAddress)
	labels = addHighCardinalityLabel(labels, supplierOperatorAddressLabelName, supplierOperatorAddress)

	// Ensure the counter is not incremented if there was an error.
	if err != nil {
		incrementAmount = 0
	}

	telemetry.IncrCounterWithLabels(
		MetricNameKeys("relays"),
		float32(incrementAmount),
		labels,
	)
}

// ClaimCounter increments a counter which tracks the number of claims at the given
// ClaimProofStage.
// If err is not nil, the counter is not incremented but Prometheus will ingest this event.
func ClaimCounter(
	claimProofStage prooftypes.ClaimProofStage,
	numClaims uint64,
	serviceId string,
	applicationAddress string,
	supplierOperatorAddress string,
	err error,
) {
	if !isTelemetyEnabled() {
		return
	}

	incrementAmount := numClaims
	labels := []metrics.Label{
		{Name: "proof_stage", Value: claimProofStage.String()},
	}

	labels = addMediumCardinalityLabel(labels, "service_id", serviceId)
	labels = addHighCardinalityLabel(labels, applicationAddressLabelName, applicationAddress)
	labels = addHighCardinalityLabel(labels, supplierOperatorAddressLabelName, supplierOperatorAddress)

	// Ensure the counter is not incremented if there was an error.
	if err != nil {
		incrementAmount = 0
	}

	telemetry.IncrCounterWithLabels(
		MetricNameKeys("claims"),
		float32(incrementAmount),
		labels,
	)
}

// RelayMiningDifficultyGauge sets a gauge which tracks the integer representation
// of the relay mining difficulty. The serviceId is used as a label to be able to
// track the difficulty for each service.
func RelayMiningDifficultyGauge(difficulty float32, serviceId string) {
	if !isTelemetyEnabled() {
		return
	}

	labels := []metrics.Label{}
	labels = addMediumCardinalityLabel(labels, "service_id", serviceId)

	telemetry.SetGaugeWithLabels(
		MetricNameKeys("relay_mining", "difficulty"),
		difficulty,
		labels,
	)
}

// RelayEMAGauge sets a gauge which tracks the relay EMA for a service.
// The serviceId is used as a label to be able to track the EMA for each service.
func RelayEMAGauge(relayEMA uint64, serviceId string) {
	if !isTelemetyEnabled() {
		return
	}

	labels := []metrics.Label{}
	labels = addMediumCardinalityLabel(labels, "service_id", serviceId)

	telemetry.SetGaugeWithLabels(
		MetricNameKeys("relay", "ema"),
		float32(relayEMA),
		labels,
	)
}

// SessionSuppliersGauge sets a gauge which tracks the number of candidates available
// for session suppliers at the given maxPerSession value.
// The serviceId is used as a label to be able to track this information for each service.
func SessionSuppliersGauge(candidates int, maxPerSession int, serviceId string) {
	if !isTelemetyEnabled() {
		return
	}

	maxPerSessionStr := strconv.Itoa(maxPerSession)
	labels := []metrics.Label{}
	labels = addMediumCardinalityLabel(labels, "service_id", serviceId)
	labels = addMediumCardinalityLabel(labels, "max_per_session", maxPerSessionStr)

	telemetry.SetGaugeWithLabels(
		MetricNameKeys("session", "suppliers"),
		float32(candidates),
		labels,
	)
}
