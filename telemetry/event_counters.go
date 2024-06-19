package telemetry

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"
)

const (
	eventTypeMetricKey = "event_type"
)

type ClaimProofStage = string

const (
	ClaimProofStageClaimed = ClaimProofStage("claimed")
	ClaimProofStageProven  = ClaimProofStage("proven")
	ClaimProofStageSettled = ClaimProofStage("settled")
	ClaimProofStageExpired = ClaimProofStage("expired")
)

type ProofRequirementReason = string

const (
	ProofNotRequired                    = ProofRequirementReason("not_required")
	ProofRequirementReasonProbabilistic = ProofRequirementReason("probabilistic_selection")
	ProofRequirementReasonThreshold     = ProofRequirementReason("above_compute_unit_threshold")
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
func ProofRequirementCounter(
	reason ProofRequirementReason,
	err error,
) {
	incrementAmount := 1
	isRequired := strconv.FormatBool(reason != ProofNotRequired)
	labels := []metrics.Label{
		{Name: "proof_required_reason", Value: reason},
		{Name: "is_required", Value: isRequired},
	}

	if err != nil {
		incrementAmount = 0
		labels = AppendErrLabels(err, labels...)
	}

	telemetry.IncrCounterWithLabels(
		[]string{eventTypeMetricKey},
		float32(incrementAmount),
		labels,
	)
}

// ClaimComputeUnitsCounter increments a counter which tracks the number of compute units
// which are represented by on-chain claims at the given ClaimProofStage.
func ClaimComputeUnitsCounter(
	claimProofStage ClaimProofStage,
	numComputeUnits uint64,
	err error,
) {
	incrementAmount := numComputeUnits
	labels := []metrics.Label{
		{Name: "unit", Value: "compute_units"},
		{Name: "claim_proof_stage", Value: claimProofStage},
	}

	// Set computeUnitsIncrementAmount to 0 if there is an error so that this counter is not incremented.
	if err != nil {
		incrementAmount = 0
		labels = AppendErrLabels(err, labels...)
	}

	telemetry.IncrCounterWithLabels(
		[]string{eventTypeMetricKey},
		float32(incrementAmount),
		labels,
	)
}

// ClaimCounter increments a counter which tracks the number of claims at the given
// ClaimProofStage.
func ClaimCounter(
	claimProofStage ClaimProofStage,
	numClaims uint64,
	err error,
) {
	incrementAmount := numClaims
	labels := []metrics.Label{
		{Name: "unit", Value: "claims"},
		{Name: "claim_proof_stage", Value: claimProofStage},
	}

	// Set incrementAmount to 0 if there is an error so that this counter is not incremented.
	if err != nil {
		incrementAmount = 0
		labels = AppendErrLabels(err, labels...)
	}

	telemetry.IncrCounterWithLabels(
		[]string{eventTypeMetricKey},
		float32(incrementAmount),
		labels,
	)
}

// TODO_IN_THIS_PR: move to labels.go & godoc
func AppendErrLabels(err error, labels ...metrics.Label) []metrics.Label {
	if err != nil {
		return append(labels, metrics.Label{Name: "error", Value: err.Error()})
	}

	return labels
}
