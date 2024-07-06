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
	// TODO_DECIDE: Decide if we want to continue using these generic metrics keys
	// or opt for specific keys for each event_type.
	// See: https://github.com/pokt-network/poktroll/pull/631#discussion_r1653760820
	eventTypeMetricKey      = "event_type"
	eventTypeMetricKeyGauge = "event_type_gauge"
)

// EventSuccessCounter increments a counter with the given data type and success status.
func EventSuccessCounter(
	eventType string,
	getValue func() float32,
	isSuccessful func() bool,
) {
	successResult := strconv.FormatBool(isSuccessful())
	value := getValue()

	telemetry.IncrCounterWithLabels(
		[]string{eventTypeMetricKey},
		value,
		[]metrics.Label{
			{Name: "type", Value: eventType},
			{Name: "is_successful", Value: successResult},
		},
	)
}

// ProofRequirementCounter increments a counter which tracks the number of claims
// which require proof for the given proof requirement reason (i.e. not required,
// probabilistic selection, above compute unit threshold).
// If err is not nil, the counter is not incremented but Prometheus will ingest this event.
func ProofRequirementCounter(
	reason prooftypes.ProofRequirementReason,
	err error,
) {
	incrementAmount := 1
	isRequired := strconv.FormatBool(reason != prooftypes.ProofRequirementReason_NOT_REQUIRED)
	labels := []metrics.Label{
		{Name: "proof_required_reason", Value: reason.String()},
		{Name: "is_required", Value: isRequired},
	}

	// Ensure the counter is not incremented if there was an error.
	if err != nil {
		incrementAmount = 0
		labels = AppendErrLabel(err, labels...)
	}

	telemetry.IncrCounterWithLabels(
		[]string{eventTypeMetricKey},
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
	err error,
) {
	incrementAmount := numComputeUnits
	labels := []metrics.Label{
		{Name: "unit", Value: "compute_units"},
		{Name: "claim_proof_stage", Value: claimProofStage.String()},
	}

	// Ensure the counter is not incremented if there was an error.
	if err != nil {
		incrementAmount = 0
		labels = AppendErrLabel(err, labels...)
	}

	telemetry.IncrCounterWithLabels(
		[]string{eventTypeMetricKey},
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
	err error,
) {
	incrementAmount := numRelays
	labels := []metrics.Label{
		{Name: "unit", Value: "relays"},
		{Name: "claim_proof_stage", Value: claimProofStage.String()},
	}

	// Ensure the counter is not incremented if there was an error.
	if err != nil {
		incrementAmount = 0
	}

	telemetry.IncrCounterWithLabels(
		[]string{eventTypeMetricKey},
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
	err error,
) {
	incrementAmount := numClaims
	labels := []metrics.Label{
		{Name: "unit", Value: "claims"},
		{Name: "claim_proof_stage", Value: claimProofStage.String()},
	}

	// Ensure the counter is not incremented if there was an error.
	if err != nil {
		incrementAmount = 0
		labels = AppendErrLabel(err, labels...)
	}

	telemetry.IncrCounterWithLabels(
		[]string{eventTypeMetricKey},
		float32(incrementAmount),
		labels,
	)
}

// RelayMiningDifficultyGauge sets a gauge which tracks the relay mining difficulty,
// which is represented by number of leading zero bits.
// The serviceId is used as a label to be able to track the difficulty for each service.
func RelayMiningDifficultyGauge(numbLeadingZeroBits int, serviceId string) {
	labels := []metrics.Label{
		{Name: "type", Value: "relay_mining_difficulty"},
		{Name: "service_id", Value: serviceId},
	}

	telemetry.SetGaugeWithLabels(
		[]string{eventTypeMetricKeyGauge},
		float32(numbLeadingZeroBits),
		labels,
	)
}

// RelayEMAGauge sets a gauge which tracks the relay EMA for a service.
// The serviceId is used as a label to be able to track the EMA for each service.
func RelayEMAGauge(relayEMA uint64, serviceId string) {
	labels := []metrics.Label{
		{Name: "type", Value: "relay_ema"},
		{Name: "service_id", Value: serviceId},
	}

	telemetry.SetGaugeWithLabels(
		[]string{eventTypeMetricKeyGauge},
		float32(relayEMA),
		labels,
	)
}

// AppendErrLabel appends a label with the name "error" and a value of the error's
// message to the given labels slice if the error is not nil.
func AppendErrLabel(err error, labels ...metrics.Label) []metrics.Label {
	if err == nil {
		return labels
	}

	return append(labels, metrics.Label{Name: "error", Value: err.Error()})
}
